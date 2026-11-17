package smapiweb

import (
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func mustWebURL(t *testing.T, raw string) {
	t.Helper()
	if raw == "" {
		return
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		t.Fatalf("%q is not an http(s) URL with a host", raw)
	}
}

// FuzzParseIndex feeds the StardewModDataset's index.
func FuzzParseIndex(f *testing.F) {
	seedFile(f, "index.json")
	f.Add([]byte(`{"A.B":["Nexus:5","Nexus:x",3],"C":null}`))
	f.Add([]byte(`{"":["Nexus:1"],"A.B":["Nexus:1","Nexus:1","CurseForge:2"]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := parseIndex(raw)
		if err != nil {
			return
		}
		for k, refs := range got {
			if k != strings.ToLower(k) {
				t.Fatalf("key %q is not lowercased", k)
			}
			if len(refs) == 0 {
				t.Fatalf("key %q has no refs", k)
			}
			if len(refs) > maxRefsPerMod {
				t.Fatalf("key %q has %d refs, cap is %d", k, len(refs), maxRefsPerMod)
			}
			seen := map[Ref]bool{}
			for _, r := range refs {
				if r.Site == "" || r.ID <= 0 {
					t.Fatalf("key %q holds invalid ref %+v", k, r)
				}
				if seen[r] {
					t.Fatalf("key %q holds duplicate %+v", k, r)
				}
				seen[r] = true
			}
		}
	})
}

// FuzzParsePage feeds a StardewModDataset mod page.
func FuzzParsePage(f *testing.F) {
	seedFile(f, "nexus-1915.json")
	f.Add([]byte(`{"Id":"2000","Downloads":[null,{"Mods":[{"Manifest":{"UniqueID":"A.B","UpdateKeys":"x"}},{"Manifest":{}},{}]}]}`))
	f.Add([]byte(`{"PageURL":"javascript:alert(1)","Downloads":[{"FileName":"a/../b.zip","Mods":[{"Manifest":{"UniqueID":"A.B"}}]}]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		p, err := parsePage(raw)
		if err != nil {
			return
		}
		dropStoragePaths(&p)
		mustWebURL(t, p.PageURL)
		for _, d := range p.Downloads {
			if d.FileName != "" && !plainFileName(d.FileName) {
				t.Fatalf("file name %q is not a plain name", d.FileName)
			}
			for _, m := range d.Mods {
				if m.UniqueID == "" {
					t.Fatal("a mod without a UniqueID was kept")
				}
				for _, dep := range m.Dependencies {
					if dep.UniqueID == "" {
						t.Fatal("a dependency without a UniqueID was kept")
					}
				}
			}
		}
	})
}

// FuzzParseCompat feeds the SMAPI compatibility list.
func FuzzParseCompat(f *testing.F) {
	f.Add([]byte(compatSample))
	f.Add([]byte(`{"mods":[{"id":[1,"a, b"],"nexus":"x"}]}`))
	f.Add([]byte(`[{"id":"A.One","nexus":10,"github":"Me/One"},{"id":"B.Two","github":"me/one"}]`))
	f.Add([]byte(`{"data":[{"id":"A.B","status":"ok","unofficialUrl":"javascript:x","successor":"https://example.com"}]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		idx, err := parseCompatJSON(raw)
		if err != nil {
			return
		}
		for k, e := range idx.ByID {
			if k == "" || k != strings.ToLower(k) {
				t.Fatalf("ByID key %q is empty or not lowercased", k)
			}
			if e.Status == "" {
				t.Fatalf("ByID %q has no status", k)
			}
			mustWebURL(t, e.UnofficialURL)
			if e.Replacement != "" {
				if u, err := url.Parse(e.Replacement); err == nil && u.Scheme != "" {
					mustWebURL(t, e.Replacement)
				}
			}
		}
		for n, id := range idx.NexusToID {
			if id == "" {
				t.Fatalf("Nexus page %d ties to no id", n)
			}
		}
		for repo, id := range idx.GitHubToID {
			if id == "" {
				t.Fatalf("repository %q ties to no id", repo)
			}
		}
		b, err := json.Marshal(idx)
		if err != nil {
			t.Fatal(err)
		}
		var back CompatIndex
		if err := json.Unmarshal(b, &back); err != nil {
			t.Fatalf("the index does not survive its own encoding: %v", err)
		}
	})
}

// FuzzUpdateAnswer feeds smapi.io's answer to an update check.
func FuzzUpdateAnswer(f *testing.F) {
	seedFile(f, "smapi-mods.json")
	f.Add([]byte(`[null,{"id":3,"suggestedUpdate":{"version":[]}}]`))
	f.Add([]byte(`[{"id":"A.B","suggestedUpdate":{"version":" ","url":"javascript:x"},"metadata":{"unofficial":{"version":"1","url":"file:///etc/passwd"}}}]`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		got, err := decodeUpdateAnswers(raw)
		if err != nil {
			return
		}
		for k, r := range got {
			if k != strings.ToLower(k) {
				t.Fatalf("key %q is not lowercased", k)
			}
			if strings.TrimSpace(r.ID) == "" {
				t.Fatalf("a result with a blank id was kept: %+v", r)
			}
			for _, u := range []*Update{r.Suggested, r.Unofficial} {
				if u != nil && strings.TrimSpace(u.Version) == "" {
					t.Fatalf("an update with a blank version was kept: %+v", r)
				}
				if u != nil {
					mustWebURL(t, u.URL)
				}
			}
		}
	})
}

// FuzzWebURL holds that a kept link is an http(s) address a URL handler can open.
func FuzzWebURL(f *testing.F) {
	f.Add("https://example.com/x")
	f.Add("\thttp://example.com\n")
	f.Add("javascript:alert(1)")
	f.Add("file:///etc/passwd")
	f.Add("https://")
	f.Add("//example.com")
	f.Fuzz(func(t *testing.T, raw string) {
		mustWebURL(t, webURL(raw))
	})
}

// FuzzBrokenOn holds that only a broken row flags a game version, and that a version string never panics.
func FuzzBrokenOn(f *testing.F) {
	f.Add("broken", "Stardew Valley 1.6", "1.5.6")
	f.Add("ok", "Stardew Valley 1.6", "1.6.0")
	f.Add("broken", "SMAPI 4.0", "")
	f.Add("broken", "Stardew Valley 1.6?", "1.6.15")
	f.Fuzz(func(t *testing.T, status, brokeIn, gameVersion string) {
		e := CompatEntry{Status: status, BrokeIn: brokeIn}
		got := e.BrokenOn(gameVersion)
		if status != StatusBroken && got {
			t.Fatalf("status %q flagged as broken on %q", status, gameVersion)
		}
	})
}
