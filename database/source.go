package database

import "github.com/fasmide/schttp/packer"

// Source is someone who receives files
// these provide their own transports
type Source interface {
	Transport() (packer.PackerCloser, error)
}
