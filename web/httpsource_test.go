package web

import "testing"

func TestDirSync(t *testing.T) {
	h := NewHTTPSource()
	h.dirSync("/hej/hej")
	if len(h.path) != 2 {
		t.Fatalf("after syncing /hej/hej path should have exactly 2 items, but had \"%+v\"", h.path)
	}
	// the path should now be [0]hej and [1]hej
	if h.path[0] != "hej" {
		t.Fatalf("zero index path was not %s: %s", "hej", "hej")
	}
}
