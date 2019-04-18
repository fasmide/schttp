// Package database provides communication and storage for scp and web packages
package database

import (
	"fmt"
	"sync"

	"github.com/teris-io/shortid"
)

var (
	lock      sync.RWMutex
	transfers map[string]Transfer
)

func init() {
	transfers = make(map[string]Transfer)
}

// ErrNotFound is returned when a transfer could not be found
type ErrNotFound error

// Get looks up a transfer and returns it
func Get(id string) (Transfer, error) {
	lock.RLock()
	t, exists := transfers[id]
	lock.RUnlock()

	if !exists {
		return nil, ErrNotFound(fmt.Errorf("transfer %s does not exist", id))
	}

	return t, nil
}

// Add generates adds a Transfer to the database and returns its id
func Add(t Transfer) (string, error) {
	id, err := shortid.Generate()
	if err != nil {
		return "", fmt.Errorf("unable to generate id: %s", err)
	}

	lock.Lock()
	transfers[id] = t
	lock.Unlock()

	return id, nil
}
