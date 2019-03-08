package scp

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/rs/xid"
	"golang.org/x/crypto/ssh"
)

type Sink struct {
	*ScpStream
	ID      xid.ID
	channel ssh.Channel
}

// NewSink returns a new initialized *Sink and prints a welcome message
func NewSink(c ssh.Channel) *Sink {

	s := &Sink{ID: xid.New(), channel: c, ScpStream: &ScpStream{Writer: c, Reader: bufio.NewReader(c)}}

	// say hello to our customer
	c.Stderr().Write([]byte(fmt.Sprintf("Velkommen, du har id %s\n", s.ID.String())))

	return s
}

// WriteTo implements the default golang WriterTo interface
// It will read the files from the remote client and pack them up in zip format
func (s *Sink) WriteTo(w io.Writer) (int64, error) {
	s.Pack(&Test{})
	w.Write([]byte("blarh blarh"))
	return 0, nil
}

type Test struct {
}

func (t *Test) File(name string, mode os.FileMode, r io.Reader) error {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("TestError: %s", err)
	}
	log.Printf("Test: read file %s, with content %s", name, d)

	return nil
}

func (t *Test) Enter(name string, mode os.FileMode) error {
	log.Printf("Test: Creating directory %s with mode %s", name, mode)
	return nil
}

func (t *Test) Leave() error {
	log.Printf("Test: Leaving")
	return nil
}
