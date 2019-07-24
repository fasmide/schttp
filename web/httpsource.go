package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"sync"

	"github.com/fasmide/schttp/packer"
)

// HTTPSource accepts files from multiple http POST requests
type HTTPSource struct {
	// this packer must not be used without getting the lock first
	packer.PackerCloser `json:"-"` // PackerCloser should not be json marshal'ed
	sync.WaitGroup      `json:"-"`
	sync.Mutex          `json:"-"`

	ID string
}

func NewHTTPSource() *HTTPSource {
	h := &HTTPSource{}

	// Add one to the waitgroup - a potential source must wait until at least
	// PackTo have been called (otherwise PackerCloser will be nill and there)
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
	// can be transfered at a time - others must wait
	h.Lock()
	n, _ := io.Copy(ioutil.Discard, rc)
	log.Printf("just discarded %s'es %d bytes", name, n)
	h.Unlock()
}
