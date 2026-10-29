package smapiweb

import (
	"encoding/json"
	"testing"
)

// Callers cache these types' JSON and read it back after upgrading, so a renamed tag would read as a zero value
// without any error. Each literal is the encoding a populated value has always had.
func TestJSONShapeIsStable(t *testing.T) {
	cases := []struct {
		name string
		in   string
		into any
	}{
		{"UpdateResult", `{"id":"A.B","known":true,"suggested":{"version":"2.0.0","url":"u","source":"Nexus:1"},` +
			`"unofficial":{"version":"2.1.0-unofficial","url":"v"},"compatibility":"Ok","compatibilitySummary":"s","brokeIn":"b","gitHubRepo":"me/ab"}`, &UpdateResult{}},
		{"CompatIndex", `{"ByID":{"a.b":{"status":"broken","summary":"s","brokeIn":"Stardew Valley 1.6","unofficialUrl":"u","unofficialVersion":"v","replacement":"r"}},` +
			`"ByNexus":{"7":{"status":"ok","summary":"","brokeIn":"","unofficialUrl":"","unofficialVersion":"","replacement":""}},` +
			`"NexusToID":{"7":"A.B"},"GitHubToID":{"me/one":"A.B"}}`, &CompatIndex{}},
		{"Ref", `{"site":"Nexus","id":1915}`, &Ref{}},
		{"Page", `{"id":1915,"name":"N","author":"P","pageUrl":"u","version":"2.9.1","downloads":[{"id":146548,"type":"Main","version":"2.8.1","fileName":"f.zip","sizeInBytes":341264,` +
			`"mods":[{"uniqueId":"A.B","name":"AB","version":"1","updateKeys":["Nexus:1915"],"dependencies":[{"uniqueId":"C.D","minimumVersion":"1.0","required":false}]}]}]}`, &Page{}},
	}
	for _, c := range cases {
		if err := json.Unmarshal([]byte(c.in), c.into); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		out, err := json.Marshal(c.into)
		if err != nil || string(out) != c.in {
			t.Errorf("%s re-encoded as\n%s\nwant\n%s\n(%v)", c.name, out, c.in, err)
		}
	}
}
