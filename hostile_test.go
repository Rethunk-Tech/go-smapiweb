package smapiweb

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

type answer struct {
	code     int
	location string
	body     []byte
}

// FuzzHostileAnswer holds that no status, Location or body makes a call panic, and that UploadLog only ever
// returns a /log/ link on the parser's own origin. One server serves the whole run: a server per input runs the
// machine out of ports.
func FuzzHostileAnswer(f *testing.F) {
	f.Add(302, "/log/abc", []byte(nil))
	f.Add(302, "//evil.example/log/abc", []byte(nil))
	f.Add(307, "/log/%2e%2e", []byte(nil))
	f.Add(200, "", []byte(`[{"id":"A","suggestedUpdate":{"version":"2","url":"javascript:x"}}]`))
	f.Add(200, "", []byte(strings.Repeat(`{"unofficial":`, 20000)))
	var a atomic.Pointer[answer]
	a.Store(&answer{code: 200})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		cur := a.Load()
		w.Header()["Location"] = []string{cur.location}
		w.WriteHeader(cur.code)
		_, _ = w.Write(cur.body)
	}))
	f.Cleanup(srv.Close)
	c := &Client{
		UpdatesURL: srv.URL + "/updates", CompatURL: srv.URL + "/compat", IndexURL: srv.URL + "/index",
		PageBase: srv.URL + "/pages", LogURL: srv.URL + "/log",
	}
	f.Fuzz(func(t *testing.T, code int, location string, body []byte) {
		if code < 200 || code > 599 || code == http.StatusNotModified {
			code = 200
		}
		a.Store(&answer{code: code, location: location, body: body})
		ctx := t.Context()
		_, _ = c.Updates(ctx, UpdateRequest{Mods: []InstalledMod{{ID: "A"}}})
		_, _ = c.CompatList(ctx)
		_, _ = c.Index(ctx)
		_, _ = c.Page(ctx, 1)
		link, err := c.UploadLog(ctx, "log", 0)
		if err == nil && !strings.HasPrefix(link, srv.URL+"/log/") {
			t.Fatalf("UploadLog gave %q for Location %q", link, location)
		}
	})
}
