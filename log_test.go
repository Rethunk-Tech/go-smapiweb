package smapiweb

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Rethunk-Tech/go-httpkit"
)

func logClient(t *testing.T, h http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), LogURL: srv.URL}
}

func TestUploadLogPostsFormAndReturnsLocation(t *testing.T) {
	var gotBody, gotType string
	c := logClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method %s", r.Method)
		}
		gotType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("Location", "/log/abc123def")
		w.WriteHeader(http.StatusFound)
	})
	link, err := c.UploadLog(t.Context(), "SMAPI 4.0 for Stardew Valley", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(link, "http://") || !strings.HasSuffix(link, "/log/abc123def") {
		t.Fatalf("link %q", link)
	}
	if !strings.HasPrefix(gotType, "application/x-www-form-urlencoded") {
		t.Fatalf("content-type %q", gotType)
	}
	vals, err := url.ParseQuery(gotBody)
	if err != nil || vals.Get("input") != "SMAPI 4.0 for Stardew Valley" {
		t.Fatalf("input %q, %v", vals.Get("input"), err)
	}
}

func TestUploadLogDoesNotModifyTheCallersClient(t *testing.T) {
	c := logClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/log/x")
		w.WriteHeader(http.StatusFound)
	})
	if c.HTTP.CheckRedirect != nil {
		t.Fatal("test client already has a policy")
	}
	if _, err := c.UploadLog(t.Context(), "log", 0); err != nil {
		t.Fatal(err)
	}
	if c.HTTP.CheckRedirect != nil {
		t.Fatal("UploadLog set a redirect policy on the caller's client")
	}
}

func TestUploadLogSendsNothingForEmptyOrOversizedText(t *testing.T) {
	var hits atomic.Int32
	c := logClient(t, func(http.ResponseWriter, *http.Request) { hits.Add(1) })
	if _, err := c.UploadLog(t.Context(), "  \n", 0); !errors.Is(err, ErrEmptyLog) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := c.UploadLog(t.Context(), "12345", 4); !errors.Is(err, ErrLogTooLarge) {
		t.Fatalf("large: %v", err)
	}
	if hits.Load() != 0 {
		t.Fatalf("%d requests sent", hits.Load())
	}
}

func TestUploadLogCapIsInclusiveAndZeroMeansNone(t *testing.T) {
	c := logClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/log/x")
		w.WriteHeader(http.StatusFound)
	})
	for _, limit := range []int{4, 0} {
		if _, err := c.UploadLog(t.Context(), "1234", limit); err != nil {
			t.Errorf("limit %d: %v", limit, err)
		}
	}
}

func TestUploadLogNeedsLogPageRedirect(t *testing.T) {
	c := logClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	if _, err := c.UploadLog(t.Context(), "not empty", 0); !errors.Is(err, ErrLogRejected) {
		t.Fatalf("a 200 is a rejection, got %v", err)
	}
}

func TestUploadLogStatusErrorIsTyped(t *testing.T) {
	c := logClient(t, func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) })
	_, err := c.UploadLog(t.Context(), "log", 0)
	se, ok := errors.AsType[*httpkit.StatusError](err)
	if !ok || se.Code != http.StatusBadGateway || se.Status != "502 Bad Gateway" {
		t.Fatalf("err = %v", err)
	}
}

func TestUploadLogRefusesARedirectAwayFromLogPages(t *testing.T) {
	for _, loc := range []string{"", "/elsewhere", "/log/", "/log/a/b"} {
		c := logClient(t, func(w http.ResponseWriter, _ *http.Request) {
			if loc != "" {
				w.Header().Set("Location", loc)
			}
			w.WriteHeader(http.StatusFound)
		})
		if _, err := c.UploadLog(t.Context(), "log", 0); !errors.Is(err, ErrNoLogLink) {
			t.Errorf("Location %q: %v", loc, err)
		}
	}
}

func TestUploadLogEscapesTheID(t *testing.T) {
	c := logClient(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Location", "/log/a%20b%3Fc")
		w.WriteHeader(http.StatusFound)
	})
	link, err := c.UploadLog(t.Context(), "log", 0)
	if err != nil || !strings.HasSuffix(link, "/log/a%20b%3Fc") {
		t.Fatalf("link %q, %v", link, err)
	}
}
