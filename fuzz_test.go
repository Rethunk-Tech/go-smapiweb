package smapiweb

import (
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
