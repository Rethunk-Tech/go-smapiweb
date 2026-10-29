package smapiweb

import (
	"os"
	"testing"
)

func fixture(t testing.TB, name string) []byte {
	t.Helper()
	root, err := os.OpenRoot("testdata")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	b, err := root.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func seedFile(f *testing.F, name string) {
	f.Helper()
	f.Add(fixture(f, name))
}
