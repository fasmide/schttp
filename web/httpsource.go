package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"sync"

	"github.com/fasmide/schttp/packer"
)

// HTTPSource accepts files from multiple http POST requests
type HTTPSource struct {
	// this packer must not be used without getting the lock first
	packer.PackerCloser `json:"-"` // PackerCloser should not be json marshal'ed
	sync.WaitGroup      `json:"-"`
	sync.Mutex          `json:"-"`

	// the current path
	path Path

	ID string
}

type Path []string

func (p Path) String() string {
	return strings.Join(p, "/")
}

func NewHTTPSource() *HTTPSource {
	h := &HTTPSource{path: make(Path, 0)}

	// Add one to the waitgroup - a potential source must wait until at least
	// PackTo have been called (otherwise PackerCloser will be nil and there
	// are no one to accept data
	h.Add(1)
	return h
}

// PackTo adds a packercloser to this source
func (h *HTTPSource) PackTo(p packer.PackerCloser) error {
	if h.PackerCloser != nil {
		return fmt.Errorf("%s already have a sink", h.ID)
	}
	h.PackerCloser = p
	h.Done()
	return nil
}

// Packer is used to fulfill database.Transfer interface and returns an
// error indicating this transfer is not able to accept files
func (h *HTTPSource) Packer() (packer.PackerCloser, error) {
	return nil, fmt.Errorf("%T cannot accept files", h)
}

// Accept accepts a POST request with a body containing a file
func (h *HTTPSource) Accept(name string, rc io.ReadCloser) {
	// wait until we can be sure the PackerCloser have been
	// set by a remote party
	h.Wait()

	// Furthermore - we must acquire a lock for this transfer - as only one file
	// can be handled at a time - others must wait
	h.Lock()
	defer h.Unlock()
	h.dirSync(name)
	n, _ := io.Copy(ioutil.Discard, rc)
	log.Printf("just discarded %s'es %d bytes", name, n)
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

	dirParts := strings.Split("/", dir)

	// first things first - if the length of where we want to go is shorter then where we are
	// we need to move up
	for len(dirParts) < len(h.path) {
		err := h.PackerCloser.Exit()
		if err != nil {
			return fmt.Errorf("could not move up from %s", h.path)
		}
		h.path = h.path[:len(h.path)-1]
	}

	// walk into the destination until it differs from h.path - count the number of levels we
	// need to move up
	for i, n := range dirParts {
		if h.path[i] == n {
			// go deeper
			continue
		}
		c := len(dirParts)
		for len(dirParts) != len(h.path) {
			err := h.PackerCloser.Exit()
			if err != nil {
				return fmt.Errorf("could not move up from %s", h.path)
			}
			h.path = h.path[:len(h.path)-1]
		}

		break

	}

	// now it should be just a matter of moving into whatever dirParts have more then where h.path already are
	for len(dirParts) != 

	return nil
}
