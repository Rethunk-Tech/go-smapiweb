package smapiweb

import (
	"bytes"
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	defaultUpdatesURL = "https://smapi.io/api/v4.0.0/mods"
	updatesBatch      = 100
	maxUpdates        = 16 << 20
)

// UpdateRequest describes the installed environment and the mods to check.
type UpdateRequest struct {
	APIVersion  string
	GameVersion string
	Platform    string // Android, Linux, Mac or Windows
	Mods        []InstalledMod
}

// InstalledMod is one mod to check.
type InstalledMod struct {
	ID         string   `json:"id"`
	UpdateKeys []string `json:"updateKeys,omitempty"`
	Version    string   `json:"installedVersion,omitempty"`
}

// Update is SMAPI's suggested newer version.
type Update struct {
	Version string `json:"version"`
	URL     string `json:"url"`
	Source  string `json:"source,omitempty"`
}

// UpdateResult is smapi.io's answer for one mod. Known is false when the mod's batch got no answer; a Known result
// with no Suggested has no update. Its JSON tags are a format contract: callers cache the encoding.
type UpdateResult struct {
	ID                   string  `json:"id"`
	Known                bool    `json:"known"`
	Suggested            *Update `json:"suggested,omitempty"`
	Unofficial           *Update `json:"unofficial,omitempty"`
	Compatibility        string  `json:"compatibility,omitempty"`
	CompatibilitySummary string  `json:"compatibilitySummary,omitempty"`
	BrokeIn              string  `json:"brokeIn,omitempty"`
	GitHubRepo           string  `json:"gitHubRepo,omitempty"`
}

type apiRequest struct {
	Mods                    []InstalledMod `json:"mods"`
	APIVersion              string         `json:"apiVersion,omitempty"`
	GameVersion             string         `json:"gameVersion,omitempty"`
	Platform                string         `json:"platform,omitempty"`
	IncludeExtendedMetadata bool           `json:"includeExtendedMetadata"`
}

type apiUpdate struct {
	Version            string     `json:"version"`
	URL                string     `json:"url"`
	UpdateKey          string     `json:"updateKey"`
	Unofficial         *apiUpdate `json:"unofficial"`
	UnofficialForSmapi *apiUpdate `json:"unofficialForSmapi"`
}

func (u *apiUpdate) asUpdate() *Update {
	if u == nil || strings.TrimSpace(u.Version) == "" {
		return nil
	}
	return &Update{Version: u.Version, URL: u.URL, Source: u.UpdateKey}
}

func firstUnofficial(parts ...*apiUpdate) *Update {
	for _, p := range parts {
		if got := p.asUpdate(); got != nil {
			return got
		}
	}
	return nil
}

type apiMod struct {
	ID              string     `json:"id"`
	SuggestedUpdate *apiUpdate `json:"suggestedUpdate"`
	Metadata        *struct {
		GitHubRepo           string     `json:"gitHubRepo"`
		CompatibilityStatus  string     `json:"compatibilityStatus"`
		CompatibilitySummary string     `json:"compatibilitySummary"`
		BrokeIn              string     `json:"brokeIn"`
		Unofficial           *apiUpdate `json:"unofficial"`
		UnofficialUpdate     *apiUpdate `json:"unofficialUpdate"`
		UnofficialForSmapi   *apiUpdate `json:"unofficialForSmapi"`
	} `json:"metadata"`
}

// Updates asks about req.Mods in batches of 100, four at a time, and asks a failed batch once more in two halves.
// Results are in request order, one per mod, and each result's ID is the request's ID, not the API's casing. A mod
// whose batch was answered is Known (a mod the API does not list is Known with no Suggested); a mod whose batch
// failed, or was not sent because an earlier batch failed, has only its ID. err is nil when every batch was
// answered; otherwise it is errors.Join of the failed batches' errors in batch order, so a caller that needs one
// error takes the first.
func (c *Client) Updates(ctx context.Context, req UpdateRequest) ([]UpdateResult, error) {
	out := make([]UpdateResult, len(req.Mods))
	for i, m := range req.Mods {
		out[i].ID = m.ID
	}
	var batches [][]InstalledMod
	for start := 0; start < len(req.Mods); start += updatesBatch {
		batches = append(batches, req.Mods[start:min(start+updatesBatch, len(req.Mods))])
	}
	answers := c.askBatches(ctx, req, batches)
	var errs []error
	for b, ans := range answers {
		switch {
		case ans.err != nil:
			errs = append(errs, ans.err)
		case ans.sent:
			for j := range batches[b] {
				i := b*updatesBatch + j
				out[i] = ans.got[strings.ToLower(req.Mods[i].ID)]
				out[i].ID = req.Mods[i].ID
				out[i].Known = true
			}
		}
	}
	return out, errors.Join(errs...)
}

type updatesAnswer struct {
	got  map[string]UpdateResult
	err  error
	sent bool
}

// updatesParallel bounds the batches asked of smapi.io at once: a cold profile of hundreds of mods is several
// batches, which one after another cost about a second.
const updatesParallel = 4

// askBatches asks smapi.io about each batch, a few at a time. Once a batch has failed, batches not yet sent are not:
// the service is down or slow, and more requests would only wait for the same timeout.
func (c *Client) askBatches(ctx context.Context, req UpdateRequest, batches [][]InstalledMod) []updatesAnswer {
	answers := make([]updatesAnswer, len(batches))
	var failed atomic.Bool
	var wg sync.WaitGroup
	slots := make(chan struct{}, updatesParallel)
	for b, asked := range batches {
		slots <- struct{}{}
		if failed.Load() {
			<-slots
			continue
		}
		wg.Go(func() {
			defer func() { <-slots }()
			got, err := c.askUpdatesRetrying(ctx, req, asked)
			answers[b] = updatesAnswer{got: got, err: err, sent: true}
			if err != nil {
				failed.Store(true)
			}
		})
	}
	wg.Wait()
	return answers
}

// askUpdatesRetrying asks once more, in two halves, when a batch fails: smapi.io sometimes runs past the request
// timeout on a full batch, and one slow answer would otherwise leave every mod in it unknown and pause all checks.
func (c *Client) askUpdatesRetrying(ctx context.Context, req UpdateRequest, mods []InstalledMod) (map[string]UpdateResult, error) {
	got, err := c.askUpdates(ctx, req, mods)
	if err == nil || ctx.Err() != nil || len(mods) < 2 {
		return got, err
	}
	half := len(mods) / 2
	first, err := c.askUpdates(ctx, req, mods[:half])
	if err != nil {
		return nil, err
	}
	second, err := c.askUpdates(ctx, req, mods[half:])
	if err != nil {
		return nil, err
	}
	maps.Copy(first, second)
	return first, nil
}

func (c *Client) askUpdates(ctx context.Context, req UpdateRequest, mods []InstalledMod) (map[string]UpdateResult, error) {
	body, err := json.Marshal(apiRequest{Mods: mods, APIVersion: req.APIVersion, GameVersion: req.GameVersion, Platform: req.Platform, IncludeExtendedMetadata: true})
	if err != nil {
		return nil, err
	}
	u := cmp.Or(c.UpdatesURL, defaultUpdatesURL)
	hr, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	hr.Header.Set("Content-Type", "application/json")
	b, err := c.do(hr, maxUpdates)
	if err != nil {
		return nil, err
	}
	return decodeUpdateAnswers(b)
}

// decodeUpdateAnswers keys SMAPI's answer by lowercased mod ID.
func decodeUpdateAnswers(b []byte) (map[string]UpdateResult, error) {
	var answers []apiMod
	if err := json.Unmarshal(b, &answers); err != nil {
		return nil, err
	}
	out := make(map[string]UpdateResult, len(answers))
	for _, a := range answers {
		r := UpdateResult{ID: a.ID, Known: true, Suggested: a.SuggestedUpdate.asUpdate()}
		if a.Metadata != nil {
			r.Compatibility = a.Metadata.CompatibilityStatus
			r.CompatibilitySummary = a.Metadata.CompatibilitySummary
			r.BrokeIn, r.GitHubRepo = a.Metadata.BrokeIn, a.Metadata.GitHubRepo
			r.Unofficial = firstUnofficial(a.Metadata.UnofficialUpdate, a.Metadata.Unofficial, a.Metadata.UnofficialForSmapi)
		}
		if r.Unofficial == nil && a.SuggestedUpdate != nil {
			r.Unofficial = firstUnofficial(a.SuggestedUpdate.Unofficial, a.SuggestedUpdate.UnofficialForSmapi)
		}
		out[strings.ToLower(a.ID)] = r
	}
	return out, nil
}
