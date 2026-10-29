package smapiweb

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// datasetServer replays the index and a few pages; any other path is a 404.
func datasetServer(t *testing.T) *Client {
	t.Helper()
	files := map[string][]byte{
		"/index":              fixture(t, "index.json"),
		"/data/1/1915.json":   fixture(t, "nexus-1915.json"),
		"/data/22/22743.json": []byte(`{"ID":22743,"Downloads":[{"ID":175656,"Type":"Main","FileName":"4a/35/fc/4a35fc45-ad1a-40a7-aa11-3eeab1cbd426"},{"ID":170000,"Type":"Main","FileName":"Alchemistry-22743-2-0-1.zip"}]}`),
		"/data/2/2000.json":   []byte(`{"Id":"2000","Name":7,"Downloads":[null,{"Id":"x","Version":1.5,"SizeInBytes":"big","Mods":[{"Manifest":{"UniqueID":"A.B","Version":2,"UpdateKeys":"Nexus:2000","Dependencies":[{"UniqueID":"C.D","IsRequired":false},{"UniqueID":"E.F","MinimumVersion":"1.0"}],"ContentPackFor":{"UniqueID":"G.H"}}},{"Manifest":{"Name":"no id"}},{}]}]}`),
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := files[r.URL.Path]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return &Client{HTTP: srv.Client(), IndexURL: srv.URL + "/index", PageBase: srv.URL + "/data"}
}

func TestIndexKeysByLowercasedID(t *testing.T) {
	got, err := datasetServer(t).Index(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	refs := got["pathoschild.contentpatcher"]
	if len(refs) != 4 || refs[0] != (Ref{Site: "CurseForge", ID: 309243}) || refs[1] != (Ref{Site: "Nexus", ID: 1915}) {
		t.Fatalf("refs = %v", refs)
	}
}

func TestParseIndexSkipsOddEntries(t *testing.T) {
	got, err := parseIndex([]byte(`{"A.B":["Nexus:5","Nexus:x","junk",3,"Nexus:6"],"C":"Nexus:1","D":null}`))
	if err != nil || len(got) != 1 || len(got["a.b"]) != 2 || got["a.b"][1].ID != 6 {
		t.Fatalf("got %v, err %v", got, err)
	}
	if _, err := parseIndex([]byte(`[]`)); err == nil {
		t.Fatal("array is not an index")
	}
}

func TestPageRealFixture(t *testing.T) {
	p, err := datasetServer(t).Page(t.Context(), 1915)
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Content Patcher" || len(p.Downloads) != 2 {
		t.Fatalf("page = %+v", p)
	}
	f := p.Downloads[1]
	if f.SizeInBytes != 389967 || f.Type != "Main" || f.Mods[0].UniqueID != "Pathoschild.ContentPatcher" || f.Mods[0].Version != "2.9.1" || f.Mods[0].UpdateKeys[0] != "Nexus:1915" {
		t.Fatalf("file = %+v", f)
	}
}

func TestPageLenientParse(t *testing.T) {
	p, err := datasetServer(t).Page(t.Context(), 2000)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Downloads) != 2 || len(p.Downloads[1].Mods) != 1 {
		t.Fatalf("downloads = %+v", p.Downloads)
	}
	m := p.Downloads[1].Mods[0]
	want := []Dependency{{"C.D", "", false}, {"E.F", "1.0", true}, {"G.H", "", true}}
	if m.Version != "2" || len(m.UpdateKeys) != 0 || len(m.Dependencies) != 3 || m.Dependencies[0] != want[0] || m.Dependencies[1] != want[1] || m.Dependencies[2] != want[2] {
		t.Fatalf("mod = %+v", m)
	}
}

func TestPageDropsStoragePathFileNames(t *testing.T) {
	p, err := datasetServer(t).Page(t.Context(), 22743)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Downloads) != 2 || p.Downloads[0].FileName != "" || p.Downloads[1].FileName != "Alchemistry-22743-2-0-1.zip" {
		t.Fatalf("downloads = %+v", p.Downloads)
	}
}

func TestPageMissingFromDatasetIsEmpty(t *testing.T) {
	p, err := datasetServer(t).Page(t.Context(), 3999)
	if err != nil || p.ID != 0 || len(p.Downloads) != 0 {
		t.Fatalf("page = %+v, err %v", p, err)
	}
}

func TestPageServerErrorIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusBadGateway) }))
	t.Cleanup(srv.Close)
	c := &Client{HTTP: srv.Client(), PageBase: srv.URL}
	if _, err := c.Page(t.Context(), 1915); err == nil {
		t.Fatal("a 502 is not an empty page")
	}
}
