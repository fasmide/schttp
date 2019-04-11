package database

import "github.com/fasmide/schttp/packer"

// Sink is someone who sends files
type Sink interface {
	TransportTo(packer.PackerCloser) error
}
