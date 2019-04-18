package packer

import (
	"io"
	"os"
)

// Packer describes an interface used by the scp package to pack
// received files
type Packer interface {
	File(string, os.FileMode, int64, io.Reader) error
	Enter(string, os.FileMode) error
	Leave() error
}

// PackerCloser embedds a close method
type PackerCloser interface {
	Packer
	Close() error
}
