package smapiweb

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDoRedirectOrigin(t *testing.T) {
	evilBody := `[{"id":"Evil.Mod","status":"ok","summary":"9.9.9","unofficialUrl":"https://malware.example/mod"}]`
	goodBody := `[{"id":"Good.Mod","status":"ok"}]`

	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(evilBody))
	}))
	defer evil.Close()

	var origin *httptest.Server
	origin = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/same":
			http.Redirect(w, r, origin.URL+"/ok", http.StatusFound)
		case "/cross":
			http.Redirect(w, r, evil.URL+"/pwn", http.StatusMovedPermanently)
		case "/ok":
			_, _ = w.Write([]byte(goodBody))
		default:
			http.NotFound(w, r)
		}
	}))
	defer origin.Close()

	c := &Client{CompatURL: origin.URL + "/same"}
	idx, err := c.CompatList(t.Context())
	if err != nil {
		t.Fatalf("same-origin redirect: %v", err)
	}
	if _, ok := idx.Lookup("Good.Mod", 0); !ok {
		t.Fatalf("same-origin redirect was not followed: %+v", idx)
	}

	c.CompatURL = origin.URL + "/cross"
	idx, err = c.CompatList(t.Context())
	var redir *CrossOriginRedirectError
	if !errors.As(err, &redir) {
		t.Fatalf("cross-origin redirect: got idx=%+v err=%v", idx, err)
	}
	if !strings.Contains(redir.URL, evil.Listener.Addr().String()) {
		t.Fatalf("error URL %q does not name the target %s", redir.URL, evil.URL)
	}
	if _, ok := idx.Lookup("Evil.Mod", 0); ok {
		t.Fatal("parsed the refused redirect body")
	}
}
