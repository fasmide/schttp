package scp

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"

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
	var bytesWritten int64

	item, err := s.Next()
	if err != nil {
		return bytesWritten, fmt.Errorf("unable to get next scp item: %s", err)
	}

	log.Printf("Item: %+v", item)
	itemData, _ := ioutil.ReadAll(item)
	log.Printf("ItemData: %s", string(itemData))
	return bytesWritten, nil
}
