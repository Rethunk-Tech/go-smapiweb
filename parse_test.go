package smapiweb

import (
	"encoding/json"
	"testing"
)

func TestParsePageDropsNonWebPageURL(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "file:///etc/passwd"} {
		p, err := parsePage([]byte(`{"PageUrl":"` + raw + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if p.PageURL != "" {
			t.Errorf("PageUrl %q came back as %q", raw, p.PageURL)
		}
	}
	p, err := parsePage([]byte(`{"PageUrl":"https://www.nexusmods.com/stardewvalley/mods/1915"}`))
	if err != nil || p.PageURL != "https://www.nexusmods.com/stardewvalley/mods/1915" {
		t.Fatalf("https PageUrl dropped: %+v, %v", p, err)
	}
}

func TestRawDependencyReadsStringIsRequired(t *testing.T) {
	for in, want := range map[string]bool{`"false"`: false, `"FALSE"`: false, `false`: false, `"true"`: true, `true`: true} {
		var d rawDependency
		if err := json.Unmarshal([]byte(`{"UniqueID":"X","IsRequired":`+in+`}`), &d); err != nil {
			t.Fatal(err)
		}
		if d.IsRequired == nil || bool(*d.IsRequired) != want {
			t.Errorf("IsRequired %s = %v, want %v", in, d.IsRequired, want)
		}
	}
}
