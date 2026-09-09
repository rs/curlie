package args

import (
	"testing"
)

func TestParse(t *testing.T) {
	expected := []string{"example.com"}
	parseAndCompare(t, []string{"curlie", "example.com"}, expected)
}

func TestParsePost(t *testing.T) {
	expected := []string{"-X", "POST", "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com"}, expected)
}

func TestParseHead(t *testing.T) {
	expected := []string{"-I", "example.com"}
	parseAndCompare(t, []string{"curlie", "head", "example.com"}, expected)
}

func parseAndCompare(t *testing.T, args, expected []string) {
	opts := Parse(args)
	if !compareStrings(opts, expected) {
		t.Errorf("Expecting %v, but got %v for %v", expected, opts, args)
	}

}

func compareStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, v := range a {
		if b[i] != v {
			return false
		}
	}
	return true
}

func TestParseJSONArrayField(t *testing.T) {
	expected := []string{"-X", "POST", "-d", `{"tables":["users"]}`, "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com", "tables[]=users"}, expected)
}

func TestParseJSONArrayFieldAppend(t *testing.T) {
	expected := []string{"-X", "POST", "-d", `{"tables":["users","admins"]}`, "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com", "tables[]=users", "tables[]=admins"}, expected)
}

func TestParseJSONArrayRaw(t *testing.T) {
	expected := []string{"-X", "POST", "-d", `{"ids":[1,2]}`, "example.com"}
	parseAndCompare(t, []string{"curlie", "post", "example.com", "ids[]:=1", "ids[]:=2"}, expected)
}
