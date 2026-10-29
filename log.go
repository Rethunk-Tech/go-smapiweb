package smapiweb

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Rethunk-Tech/go-httpkit"
)

const (
	defaultLogURL = "https://smapi.io/log"
	// uploadTimeout is longer than requestTimeout: the body is a whole log, up to megabytes, and the parser reads
	// all of it before it redirects.
	uploadTimeout = 60 * time.Second
	// maxLogAnswer bounds how much of the answer's body is read before it is dropped; only the redirect matters.
	maxLogAnswer = 1 << 20
)

var (
	// ErrEmptyLog is UploadLog's answer to text with nothing in it. No request is sent.
	ErrEmptyLog = errors.New("the log is empty")
	// ErrLogTooLarge is UploadLog's answer to text over maxBytes. No request is sent.
	ErrLogTooLarge = errors.New("the log is too large to upload")
	// ErrLogRejected means the parser answered 200 with its page instead of a redirect, which is how it refuses a log.
	ErrLogRejected = errors.New("smapi.io did not accept the log")
	// ErrNoLogLink means the parser redirected somewhere that is not a /log/<id> page.
	ErrNoLogLink = errors.New("smapi.io did not return a log link")
)

// UploadLog posts text to the SMAPI log parser as the form field input and returns the public link,
// https://<host>/log/<id>. Success is a redirect to /log/<id>; the parser answers a failed or empty upload with
// HTTP 200 and its page, which is ErrLogRejected. Any other status is a *httpkit.StatusError. maxBytes caps
// len(text); 0 means no cap. The link is rebuilt from the redirect's scheme and host with the id path-escaped.
func (c *Client) UploadLog(ctx context.Context, text string, maxBytes int) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", ErrEmptyLog
	}
	if maxBytes > 0 && len(text) > maxBytes {
		return "", ErrLogTooLarge
	}
	ctx, cancel := context.WithTimeout(ctx, uploadTimeout)
	defer cancel()
	form := url.Values{"input": {text}}.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cmp.Or(c.LogURL, defaultLogURL), strings.NewReader(form))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpkit.WithRedirect(c.HTTP, httpkit.NoFollow).Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, maxLogAnswer))
	if res.StatusCode < 300 || res.StatusCode > 399 {
		if res.StatusCode == http.StatusOK {
			return "", ErrLogRejected
		}
		return "", &httpkit.StatusError{Code: res.StatusCode, Status: res.Status}
	}
	loc := res.Header.Get("Location")
	if loc == "" {
		return "", fmt.Errorf("%w (HTTP %s)", ErrNoLogLink, res.Status)
	}
	abs, err := res.Request.URL.Parse(loc)
	if err != nil {
		return "", err
	}
	id, ok := strings.CutPrefix(abs.Path, "/log/")
	if !ok || id == "" || strings.Contains(id, "/") {
		return "", fmt.Errorf("%w (redirected to %q)", ErrNoLogLink, abs.Path)
	}
	return abs.Scheme + "://" + abs.Host + "/log/" + url.PathEscape(id), nil
}
