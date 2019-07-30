package web

import (
	"io"
	"os"
	"testing"
)

func TestDirSync(t *testing.T) {
	h := NewHTTPSource()

	// hackish way to get around httpsource'es locking
	go h.PackTo(&packerFixture{})
	h.Close()

	h.dirSync("/hej/hej")
	if h.path.String() != "/hej/" {
		t.Fatalf("after syncing /hej/hej path should should be \"/hej/\" but was \"%s\"", h.path.String())
	}
	// the path should now be [0]hej and [1]hej
	if h.path[0] != "hej" {
		t.Fatalf("zero index path was not %s: %s", "hej", h.path[0])
	}

	h.dirSync("/hello/world/world")
	if h.path[0] != "hello" && h.path[1] != "world" {
		t.Fatalf("h.path is wrong: %s", h.path)
	}

	h.dirSync("/files/tekst.txt")
	if h.path[0] != "files" {
		t.Fatalf("h.path is wrong: %s, should be \"/files/\"", h.path)
	}

	h.dirSync("/tekst.fil")
	if h.path.String() != "/" {
		t.Fatalf("h.path wrong %s, should be just /", h.path)
	}

	h.dirSync("/really/deep/stuff/here/be/file.txt")
	if h.path.String() != "/really/deep/stuff/here/be/" {
		t.Fatalf("h.path wrong %s, should be /really/deep/stuff/here/be/", h.path)
	}

	h.dirSync("/really/another.txt")
	if h.path.String() != "/really/" {
		t.Fatalf("h.path wrong %s, should be /really/", h.path)
	}
}

type packerFixture struct {
}

func (p *packerFixture) Enter(name string, _ os.FileMode) error {
	return nil
}

func (p *packerFixture) Leave() error {
	return nil
}

func (p *packerFixture) File(_ string, _ os.FileMode, _ int64, _ io.Reader) error {
	return nil
}

func (p *packerFixture) Close() error {
	return nil
}

func (p *packerFixture) Exit() error {
	return nil
}
