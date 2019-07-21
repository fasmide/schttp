// Package database provides communication and storage for scp and web packages
package database

import (
	"fmt"
	"sync"

	"github.com/fasmide/schttp/packer"
	"github.com/teris-io/shortid"
)

func init() {
	transfers = make(map[string]Transfer)
}

var lock sync.Mutex
var transfers map[string]Transfer

// Add adds a transfer and returns its id
func Add(t Transfer) (string, error) {
	id, err := shortid.Generate()
	if err != nil {
		return "", err
	}

	lock.Lock()
	transfers[id] = t
	lock.Unlock()

	return id, nil
}

// Fetch fetches and removes a transfer from the database
func Fetch(id string) (Transfer, error) {
	lock.Lock()
	defer lock.Unlock()

	t, exists := transfers[id]
	if !exists {
		return nil, fmt.Errorf("No transfer with id %s", id)
	}

	delete(transfers, id)
	return t, nil

}

// Shutdown tells waiting transfers they need to reconnect
func Shutdown(msg string) {
	// TODO: handle shutdowns somehow
}

// Transfer does not care about direction
// it is up to the transfer it self to return an error
// if it is not able to accept or provide files
type Transfer interface {
	// Packer is used when sending files
	// i.e. the transfer will provide a packer
	// which the other end will put files and folders into
	Packer() (packer.PackerCloser, error)

	// PackTo is used when providing a packercloser to receive files
	// i.e. you provide a packer which the other end will pack files
	// and folders into
	PackTo(packer.PackerCloser) error
}
