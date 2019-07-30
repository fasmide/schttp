package web

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/fasmide/schttp/packer"
)

// HTTPSource accepts files from multiple http POST requests
type HTTPSource struct {
	// this packer must not be used without getting the lock first
	packer.PackerCloser `json:"-"` // PackerCloser should not be json marshal'ed
	waitPackerCloser    sync.WaitGroup
	sync.Mutex          `json:"-"`

	waitFinished sync.WaitGroup

	// the current path
	path Path

	ID string
}

type Path []string

func (p Path) String() string {
	if len(p) == 0 {
		return "/"
	}
	return fmt.Sprintf("/%s/", strings.Join(p, "/"))
}

func NewHTTPSource() *HTTPSource {
	h := &HTTPSource{path: make(Path, 0)}

	// Add one to the waitgroup - a potential source must wait until at least
	// PackTo have been called (otherwise PackerCloser will be nil and there
	// are no one to accept data
	h.waitPackerCloser.Add(1)
	return h
}

// PackTo adds a packercloser to this source
// it will also block until we are finished
func (h *HTTPSource) PackTo(p packer.PackerCloser) error {
	if h.PackerCloser != nil {
		return fmt.Errorf("%s already have a sink", h.ID)
	}
	h.PackerCloser = p

	// indicate to potential sources that a sink have arrived to accept data
	h.waitPackerCloser.Done()

	// wait for source to indicate no more data is coming
	h.waitFinished.Add(1)
	h.waitFinished.Wait()

	return nil
}

// Packer is used to fulfill database.Transfer interface and returns an
// error indicating this transfer is not able to accept files
func (h *HTTPSource) Packer() (packer.PackerCloser, error) {
	return nil, fmt.Errorf("%T cannot accept files", h)
}

// Accept accepts a POST request with a body containing a file
func (h *HTTPSource) Accept(name string, size int64, r io.Reader) error {
	// wait until we can be sure the PackerCloser have been
	// set by a remote party
	h.waitPackerCloser.Wait()

	// Furthermore - we must acquire a lock for this transfer - as only one file
	// can be handled at a time - others must wait
	h.Lock()
	defer h.Unlock()

	err := h.dirSync(name)
	if err != nil {
		return fmt.Errorf("could not synchronize directories: %s", err)
	}
	_, filename := path.Split(name)
	err = h.File(filename, os.FileMode(0664), size, r)
	if err != nil {
		return fmt.Errorf("could not send file: %s", err)
	}
	return nil
}

func (h *HTTPSource) Close() error {
	// wait for a packerCloser in the unlikly event a user closes his source before a
	// sink have turned up
	h.waitPackerCloser.Wait()

	err := h.PackerCloser.Close()
	if err != nil {
		return fmt.Errorf("unable to close packer: %s", err)
	}

	h.waitFinished.Done()
	return nil
}

// dirSync takes a directory where the next file should be placed
// and walks around using PackerCloser's Enter and Exit functions
func (h *HTTPSource) dirSync(incoming string) error {
	cleaned := path.Clean(incoming)
	dir, _ := path.Split(cleaned)

	// dont do anything if we are already at the right location
	if h.path.String() == dir {
		return nil
	}

	f := func(c rune) bool {
		return c == '/'
	}
	dirParts := strings.FieldsFunc(dir, f)

	// we need to figure out how many levels to exit
	// we do this by looking at our own path and match every index in the destination path
	// until they stop to match, then look at how meny extra levels are present in our own path and move up from them
	// should we find that the current index does not exist in the destination path - we just need to level up dirs and then we are done
	levelsToExit := 0
	for i, p := range h.path {
		if i >= len(dirParts) {
			// if we are in a deeper level then our destination - we should level up
			levelsToExit = len(h.path) - len(dirParts)
			break
		}

		if p != dirParts[i] {
			// if the current index does not match the destination index - this is where we need to level up from
			levelsToExit = len(h.path) - i
			break
		}
	}

	// move up levelsToExit' times
	for index := 0; index < levelsToExit; index++ {
		err := h.Exit()
		if err != nil {
			return fmt.Errorf("could not exit %s: %s", h.path, err)
		}
		h.path = h.path[:len(h.path)-1]
	}

	// if dirParts is longer we should move in
	for index := len(h.path); len(dirParts) > len(h.path); index++ {
		err := h.Enter(dirParts[index], os.FileMode(0664))
		if err != nil {
			return fmt.Errorf("could not enter %s: %s", dirParts[index], err)
		}
		h.path = append(h.path, dirParts[index])
	}

	return nil
}
