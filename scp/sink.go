package scp

import (
	"bufio"
	"fmt"
	"io"

	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
)

type Sink struct {
	// TODO: embedd noget der kan forstå scp wire protokolen
	ID      xid.ID
	channel ssh.Channel
	scanner *bufio.Scanner
}

// NewSink returns a new initialized *Sink and prints a welcome message
func NewSink(c ssh.Channel) *Sink {

	s := &Sink{ID: xid.New(), channel: c, scanner: bufio.NewScanner(c)}

	// say hello to our customer
	c.Stderr().Write([]byte(fmt.Sprintf("Velkommen, du har id %s\n", s.ID.String())))

	return s
}

// WriteTo implements the default golang WriterTo interface
// It will read the files from the remote client and pack them up in zip format
func (s *Sink) WriteTo(w io.Writer) (int64, error) {
	var bytesWritten int64
	s.channel.Write([]byte{0x00})

	for s.scanner.Scan() {
		n, err := w.Write(s.scanner.Bytes())
		if err != nil {
			return bytesWritten, err
		}
		bytesWritten += int64(n)

		// ask for more
		s.channel.Write([]byte{0x00})
	}
	if err := s.scanner.Err(); err != nil {
		return bytesWritten, err
	}
	s.channel.Close()
	return bytesWritten, nil
}
