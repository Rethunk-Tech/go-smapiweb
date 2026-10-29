package smapiweb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

var cpRequest = UpdateRequest{APIVersion: "4.5.2", GameVersion: "1.6.15", Platform: "Windows", Mods: []InstalledMod{
	{ID: "Pathoschild.ContentPatcher", UpdateKeys: []string{"Nexus:1915"}, Version: "2.0.0"},
	{ID: "Nope.Nothing", Version: "1.0"},
}}

func updatesClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), UpdatesURL: srv.URL}
}

func TestUpdatesRealAnswer(t *testing.T) {
	body := fixture(t, "smapi-mods.json")
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write(body) })
	got, err := c.Updates(t.Context(), cpRequest)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !got[0].Known || got[0].Suggested == nil || got[0].Suggested.Version != "2.9.1" ||
		got[0].Compatibility != "Ok" || got[0].GitHubRepo != "Pathoschild/StardewMods" {
		t.Fatalf("got %+v", got)
	}
	if !got[1].Known || got[1].Suggested != nil || got[1].ID != "Nope.Nothing" {
		t.Fatalf("a mod with no update is known: %+v", got[1])
	}
}

func TestUpdatesReadsCompatibilitySummary(t *testing.T) {
	t.Parallel()
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
		  {"id":"Old.Mod","metadata":{
		    "compatibilityStatus":"Abandoned",
		    "compatibilitySummary":"use [New Mod](https://www.nexusmods.com/stardewvalley/mods/5098) instead.",
		    "brokeIn":""
		  }}
		]`))
	})
	got, err := c.Updates(t.Context(), UpdateRequest{Mods: []InstalledMod{{ID: "Old.Mod"}}})
	if err != nil || len(got) != 1 || !got[0].Known || got[0].Compatibility != "Abandoned" ||
		got[0].CompatibilitySummary != "use [New Mod](https://www.nexusmods.com/stardewvalley/mods/5098) instead." {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestUpdatesReadsUnofficialFields(t *testing.T) {
	t.Parallel()
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[
		  {"id":"me.a","suggestedUpdate":{"version":"2.0.0","url":"https://n.test/a"},
		    "metadata":{"unofficialUpdate":{"version":"2.1.0-unofficial.1-x","url":"https://smapi.io/u"}}},
		  {"id":"me.b","suggestedUpdate":{"version":"1.1.0","url":"https://n.test/b","unofficialForSmapi":{"version":"1.2.0-unofficial","url":"https://smapi.io/b"}},
		    "metadata":{}},
		  {"id":"me.c","metadata":{"unofficial":{"version":"3.0.0-unofficial","url":"https://smapi.io/c"}}},
		  {"id":"me.d","suggestedUpdate":{"version":"  "}}
		]`))
	})
	got, err := c.Updates(t.Context(), UpdateRequest{Mods: []InstalledMod{{ID: "me.a"}, {ID: "me.b"}, {ID: "me.c"}, {ID: "me.d"}}})
	if err != nil || len(got) != 4 || got[0].Suggested == nil || got[0].Suggested.Version != "2.0.0" ||
		got[0].Unofficial == nil || got[0].Unofficial.Version != "2.1.0-unofficial.1-x" ||
		got[1].Unofficial == nil || got[1].Unofficial.Version != "1.2.0-unofficial" ||
		got[2].Unofficial == nil || got[2].Unofficial.Version != "3.0.0-unofficial" ||
		got[3].Suggested != nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestUpdatesMatchesIDsIgnoringCaseAndReturnsTheRequestsID(t *testing.T) {
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"ME.A","suggestedUpdate":{"version":"2.0.0","url":"u"}}]`))
	})
	got, err := c.Updates(t.Context(), UpdateRequest{Mods: []InstalledMod{{ID: "me.a"}}})
	if err != nil || got[0].ID != "me.a" || got[0].Suggested == nil {
		t.Fatalf("got %+v, %v", got, err)
	}
}

func TestUpdatesBatchesAndSendsRequest(t *testing.T) {
	var batches atomic.Int32
	var mu sync.Mutex
	var first apiRequest
	c := updatesClient(t, func(w http.ResponseWriter, r *http.Request) {
		var req apiRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Error(err)
		}
		batches.Add(1)
		// Batches are asked concurrently, so any full one stands for the first.
		if len(req.Mods) == updatesBatch {
			mu.Lock()
			first = req
			mu.Unlock()
		}
		out := make([]apiMod, len(req.Mods))
		for i, m := range req.Mods {
			out[i] = apiMod{ID: m.ID}
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	req := UpdateRequest{APIVersion: "4.5.2", GameVersion: "1.6.15", Platform: "Linux"}
	for i := range 230 {
		req.Mods = append(req.Mods, InstalledMod{ID: "M" + string(rune('A'+i%26)) + string(rune('a'+i/26)), Version: "1.0"})
	}
	got, err := c.Updates(t.Context(), req)
	if err != nil || batches.Load() != 3 || len(got) != 230 || !got[229].Known || got[229].ID != req.Mods[229].ID {
		t.Fatalf("batches %d, results %d, err %v", batches.Load(), len(got), err)
	}
	if len(first.Mods) != 100 || !first.IncludeExtendedMetadata || first.Platform != "Linux" || first.GameVersion != "1.6.15" || first.APIVersion != "4.5.2" {
		t.Fatalf("request = %+v", first)
	}
}

func TestAFailedBatchIsAskedAgainInHalves(t *testing.T) {
	var asked atomic.Int32
	c := updatesClient(t, func(w http.ResponseWriter, r *http.Request) {
		if asked.Add(1) == 1 {
			http.Error(w, "busy", http.StatusBadGateway)
			return
		}
		var body apiRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		out := make([]map[string]string, 0, len(body.Mods))
		for _, m := range body.Mods {
			out = append(out, map[string]string{"id": m.ID})
		}
		_ = json.NewEncoder(w).Encode(out)
	})
	got, err := c.Updates(t.Context(), UpdateRequest{Mods: []InstalledMod{{ID: "A.Mod", Version: "1"}, {ID: "B.Mod", Version: "1"}, {ID: "C.Mod", Version: "1"}}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range got {
		if !r.Known {
			t.Fatalf("%s is unknown after the retry: %+v", r.ID, got)
		}
	}
	if n := asked.Load(); n != 3 {
		t.Errorf("asked %d times, want the failed batch then its two halves", n)
	}
}

func TestAFailedBatchKeepsTheAnsweredOnesAndErrorsWithAJoin(t *testing.T) {
	var asked atomic.Int32
	c := updatesClient(t, func(w http.ResponseWriter, r *http.Request) {
		var body apiRequest
		_ = json.NewDecoder(r.Body).Decode(&body)
		asked.Add(1)
		// The first batch's mods are the only ones answered; everything else is a 502.
		if body.Mods[0].ID != "M0" {
			http.Error(w, "down", http.StatusBadGateway)
			return
		}
		_ = json.NewEncoder(w).Encode([]map[string]string{{"id": "M0"}})
	})
	// Five batches, so the fifth waits for a free slot.
	req := UpdateRequest{}
	for i := range 5 * updatesBatch {
		req.Mods = append(req.Mods, InstalledMod{ID: "M" + string(rune('0'+i/updatesBatch)) + "-" + string(rune('a'+i%26))})
	}
	req.Mods[0].ID = "M0"
	got, err := c.Updates(t.Context(), req)
	if err == nil {
		t.Fatal("want the failed batches' error")
	}
	var uerr interface{ Unwrap() []error }
	if !errors.As(err, &uerr) || len(uerr.Unwrap()) == 0 {
		t.Fatalf("want a joined error, got %T %v", err, err)
	}
	if len(got) != len(req.Mods) || !got[0].Known || got[0].ID != "M0" {
		t.Fatalf("the answered batch keeps its answers: %+v", got[0])
	}
	if got[len(got)-1].Known || got[len(got)-1].ID != req.Mods[len(req.Mods)-1].ID {
		t.Fatalf("a failed batch has only its ID: %+v", got[len(got)-1])
	}
}

func TestUpdatesJoinedErrorCarriesTheTransportError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	dead := srv.URL
	srv.Close()
	c := &Client{UpdatesURL: dead}
	got, err := c.Updates(t.Context(), cpRequest)
	if _, ok := errors.AsType[*url.Error](err); !ok {
		t.Fatalf("err = %T %v, want a *url.Error inside", err, err)
	}
	if len(got) != 2 || got[0].Known || got[0].ID != cpRequest.Mods[0].ID {
		t.Fatalf("got %+v", got)
	}
}

func TestUpdatesWithNoModsAsksNothing(t *testing.T) {
	c := updatesClient(t, func(http.ResponseWriter, *http.Request) { t.Error("a request was sent") })
	got, err := c.Updates(t.Context(), UpdateRequest{})
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v", got, err)
	}
}

func TestUpdatesStopsRetryingOnceTheContextIsDone(t *testing.T) {
	var asked atomic.Int32
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) {
		asked.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusBadGateway)
	})
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()
	_, err := c.Updates(ctx, cpRequest)
	if err == nil || asked.Load() > 1 {
		t.Fatalf("asked %d times, err %v", asked.Load(), err)
	}
}
