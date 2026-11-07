package smapiweb

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestUploadLogRefusesAnAbsoluteRedirectToAnotherSite(t *testing.T) {
	t.Parallel()
	for _, loc := range []string{"file:///log/abc", "http://attacker.example/log/abc", "https://127.0.0.1/log/abc"} {
		c := logClient(t, func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", loc)
			w.WriteHeader(http.StatusFound)
		})
		if link, err := c.UploadLog(t.Context(), "log", 0); err == nil {
			t.Errorf("Location %q gave link %q", loc, link)
		}
	}
}

func TestWebURLKeepsOnlyHTTPAddresses(t *testing.T) {
	t.Parallel()
	for raw, want := range map[string]string{
		"https://example.com/x": "https://example.com/x", "\thttp://example.com\n": "http://example.com",
		"javascript:alert(1)": "", "file:///etc/passwd": "", "https://": "", "//example.com": "", "": "",
	} {
		if got := webURL(raw); got != want {
			t.Errorf("webURL(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestUpdateAnswersDropNonWebURLs(t *testing.T) {
	t.Parallel()
	c := updatesClient(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`[{"id":"A.B","suggestedUpdate":{"version":"9.9.9","url":"javascript:alert(1)"},` +
			`"metadata":{"unofficialUpdate":{"version":"1.0","url":"file:///etc/passwd"}}}]`))
	})
	got, err := c.Updates(t.Context(), UpdateRequest{Mods: []InstalledMod{{ID: "A.B"}}})
	if err != nil || len(got) != 1 || got[0].Suggested == nil || got[0].Suggested.URL != "" ||
		(got[0].Unofficial != nil && got[0].Unofficial.URL != "") {
		t.Fatalf("got %+v, %v", got, err)
	}
}

// A 307 re-sends the POST body, the installed-mod list, so it must not leave the origin.
func TestUpdatesDoNotResendTheBodyToAnotherOrigin(t *testing.T) {
	t.Parallel()
	var leaked atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { leaked.Add(1) }))
	t.Cleanup(other.Close)
	c := updatesClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusTemporaryRedirect)
	})
	if _, err := c.Updates(t.Context(), cpRequest); err == nil {
		t.Fatal("a redirected update ask succeeded")
	}
	if leaked.Load() != 0 {
		t.Fatal("the installed-mod list reached another origin")
	}
}

func TestPlainFileNameRefusesDotEntriesColonsAndControls(t *testing.T) {
	t.Parallel()
	for name, want := range map[string]bool{
		"mod-1.zip": true, "..": false, ".": false, "a/b": false, `a\b`: false, "C:evil": false, "a\x00b": false, "a\nb": false,
	} {
		if got := plainFileName(name); got != want {
			t.Errorf("plainFileName(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestParseIndexBoundsAndDedupesRefsPerMod(t *testing.T) {
	t.Parallel()
	var keys []string
	for i := range 3 * maxRefsPerMod {
		keys = append(keys, `"Nexus:`+strconv.Itoa(i%(2*maxRefsPerMod))+`"`, `"Nexus:1"`)
	}
	idx, err := parseIndex([]byte(`{"A.B":[` + strings.Join(keys, ",") + `]}`))
	if err != nil || len(idx["a.b"]) != maxRefsPerMod {
		t.Fatalf("len %d, %v", len(idx["a.b"]), err)
	}
	seen := map[Ref]bool{}
	for _, r := range idx["a.b"] {
		if seen[r] {
			t.Fatalf("duplicate %+v", r)
		}
		seen[r] = true
	}
}
