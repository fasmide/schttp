package scp

import (
	"io"

	"github.com/teris-io/shortid"

	"golang.org/x/crypto/ssh"
)

type Source struct {
	ID      string
	channel ssh.Channel
}

func NewSource(c ssh.Channel) (*Source, error) {
	id, err := shortid.Generate()
	if err != nil {
		return nil, err
	}

	return &Source{ID: id, channel: c}, nil
}

func (s *Source) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}
