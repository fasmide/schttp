package scp

import (
	"io"

	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
)

type Source struct {
	ID      xid.ID
	channel ssh.Channel
}

func NewSource(c ssh.Channel) *Source {
	return &Source{ID: xid.New(), channel: c}
}

func (s *Source) ReadFrom(r io.Reader) (int64, error) {
	return 0, nil
}
