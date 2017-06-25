package panuxpackager

import (
	"encoding/json"
	"testing"
)

func TestParseRaw(t *testing.T) {
	teststring := `
        tools:
            - Apple
            - Pear
        version: v1.0
        sources:
            - https://example.com/source1
            - https://example.com/source2
        script:
            - echo "Hello world"
            - cat example.txt
    `
	pg, err := ParseRaw([]byte(teststring))
	if err != nil {
		t.Fatal(err)
	}
	expected := RawPackageGenerator{
		Tools: []string{
			"Apple",
			"Pear",
		},
		Version: "v1.0",
		Sources: []string{
			"https://example.com/source1",
			"https://example.com/source2",
		},
		Script: []string{
			"echo \"Hello world\"",
			"cat example.txt",
		},
	}
	a, err := json.Marshal(pg)
	if err != nil {
		t.Fatal("Failed to check results")
	}
	b, err := json.Marshal(expected)
	if err != nil {
		t.Fatal("Failed to check results")
	}
	if string(a) != string(b) {
		t.Fatal("incorrect unmarshal")
	}
}

func TestBadParse(t *testing.T) {
	pg, err := ParseRaw([]byte("This should return an error"))
	if err == nil {
		t.Fatal("Test should have returned error but did not")
	}
	if pg.Script != nil || pg.Sources != nil || pg.Tools != nil || pg.Version != "" {
		t.Fatal("Did not return empty RawPackageGenerator on error")
	}
}
