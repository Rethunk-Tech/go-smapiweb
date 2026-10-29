package smapiweb

import (
	"encoding/json"
	"strings"
	"testing"
)

// FuzzParseIndex feeds the StardewModDataset's index.
func FuzzParseIndex(f *testing.F) {
	seedFile(f, "index.json")
	f.Add([]byte(`{"A.B":["Nexus:5","Nexus:x",3],"C":null}`))
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
		}
	})
}

// FuzzParsePage feeds a StardewModDataset mod page.
func FuzzParsePage(f *testing.F) {
	seedFile(f, "nexus-1915.json")
	f.Add([]byte(`{"Id":"2000","Downloads":[null,{"Mods":[{"Manifest":{"UniqueID":"A.B","UpdateKeys":"x"}},{"Manifest":{}},{}]}]}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		p, _ := parsePage(raw)
		dropStoragePaths(&p)
		for _, d := range p.Downloads {
			if strings.ContainsAny(d.FileName, `/\`) {
				t.Fatalf("file name %q holds a path separator", d.FileName)
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
	f.Fuzz(func(t *testing.T, raw []byte) {
		idx, err := parseCompatJSON(raw)
		if err != nil {
			return
		}
		for k := range idx.ByID {
			if k == "" || k != strings.ToLower(k) {
				t.Fatalf("ByID key %q is empty or not lowercased", k)
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
