package web

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/fasmide/schttp/packer"
)

// HTTPSource accepts files from multiple http POST requests
type HTTPSource struct {
	// this packer must not be used without getting the lock first
	packer.PackerCloser `json:"-"` // PackerCloser should not be json marshal'ed
	sync.Mutex

	ID string
}

// PackTo adds a packercloser to this source
func (h *HTTPSource) PackTo(p packer.PackerCloser) error {
	if h.PackerCloser != nil {
		return fmt.Errorf("%s already have a sink", h.ID)
	}
	h.PackerCloser = p
	return nil
}

// Packer is used to fulfill database.Transfer interface and returns an
// error indicating this transfer is not able to accept files
func (h *HTTPSource) Packer() (packer.PackerCloser, error) {
	return nil, fmt.Errorf("%T cannot accept files", h)
}

// Accept accepts a POST request with a body containing a file
func (h *HTTPSource) Accept(r *http.Request) {

}
