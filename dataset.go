package smapiweb

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/Rethunk-Tech/go-httpkit"
)

const (
	defaultIndexURL = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/indexes/pages%20by%20mod%20ID.json"
	defaultPageBase = "https://raw.githubusercontent.com/Pathoschild/StardewModDataset/main/dataset/data/Nexus"
	maxIndex        = 32 << 20
	maxPage         = 8 << 20
)

// Ref names a mod page: Site is "Nexus" or "CurseForge", as the dataset names it. Its JSON tags are a format
// contract: callers cache the encoding.
type Ref struct {
	Site string `json:"site"`
	ID   int    `json:"id"`
}

// Page is one Nexus mod page's current files as the dataset records them.
type Page struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Author    string `json:"author"`
	PageURL   string `json:"pageUrl"`
	Version   string `json:"version"`
	Downloads []File `json:"downloads"`
}

// File is one downloadable file; its Type is Nexus' category (Main, Optional, ...). FileName is empty when the
// dataset gave a storage path ("4a/35/fc/<uuid>"), which names nothing a user or Nexus would recognise.
type File struct {
	ID          int64  `json:"id"`
	Type        string `json:"type"`
	Version     string `json:"version"`
	FileName    string `json:"fileName"`
	SizeInBytes int64  `json:"sizeInBytes"`
	Mods        []Mod  `json:"mods"`
}

// Mod is a SMAPI manifest found inside a file.
type Mod struct {
	UniqueID     string       `json:"uniqueId"`
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	UpdateKeys   []string     `json:"updateKeys"`
	Dependencies []Dependency `json:"dependencies"`
}

// Dependency is a Dependencies[] entry, or the ContentPackFor framework as a required one.
type Dependency struct {
	UniqueID       string `json:"uniqueId"`
	MinimumVersion string `json:"minimumVersion"`
	Required       bool   `json:"required"`
}

// Index returns the dataset's index of which pages ship which mod, keyed by lowercased UniqueID. Entries that are
// not "Site:number" are skipped; an index with no usable entry is an error.
func (c *Client) Index(ctx context.Context) (map[string][]Ref, error) {
	u := cmp.Or(c.IndexURL, defaultIndexURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	b, err := c.do(req, maxIndex)
	if err != nil {
		return nil, err
	}
	return parseIndex(b)
}

// parseIndex keys the index by lowercased UniqueID and skips entries that are not "Site:number".
func parseIndex(b []byte) (map[string][]Ref, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := make(map[string][]Ref, len(raw))
	for id, v := range raw {
		var keys list
		if json.Unmarshal(v, &keys) != nil {
			continue
		}
		key := strings.ToLower(id)
		for _, k := range keys {
			site, num, ok := strings.Cut(string(k), ":")
			n, err := strconv.Atoi(num)
			if !ok || err != nil || site == "" {
				continue
			}
			out[key] = append(out[key], Ref{Site: site, ID: n})
		}
	}
	if len(out) == 0 {
		return nil, errors.New("mod dataset index is empty")
	}
	return out, nil
}

// Page returns Nexus mod page id's files. A page the dataset lacks is an empty Page and a nil error: the dataset
// answered, it just does not know the page.
func (c *Client) Page(ctx context.Context, id int) (Page, error) {
	base := cmp.Or(c.PageBase, defaultPageBase)
	u, err := url.JoinPath(base, url.PathEscape(strconv.Itoa(id/1000)), url.PathEscape(strconv.Itoa(id)+".json"))
	if err != nil {
		return Page{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return Page{}, err
	}
	b, err := c.do(req, maxPage)
	if errors.Is(err, httpkit.ErrNotFound) {
		return Page{}, nil
	}
	if err != nil {
		return Page{}, err
	}
	p, err := parsePage(b)
	dropStoragePaths(&p)
	return p, err
}

// dropStoragePaths blanks a file name that is a storage path: it is never a name to save a file under.
func dropStoragePaths(p *Page) {
	for i := range p.Downloads {
		if strings.ContainsAny(p.Downloads[i].FileName, `/\`) {
			p.Downloads[i].FileName = ""
		}
	}
}
