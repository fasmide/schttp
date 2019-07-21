package scp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"path"

	"github.com/fasmide/schttp/database"
	"github.com/fasmide/schttp/packer"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

// Sink is a client that sends files into schttp (schttp sinks files)
type Sink struct {
	*ScpStream
	ID      string
	channel ssh.Channel
}

// SinkBanner is printed out when ready to stream files
const SinkBanner = `    -----------------------

    One time urls for download
      %s.zip 
    or
      %s.tar.gz

    Or unpack directly on another box:
      curl %s.tar.gz | tar xvz
    (May overwrite existing files)
`

// NewSink returns a new initialized *Sink and prints a welcome message
func NewSink(c ssh.Channel) (*Sink, error) {

	s := &Sink{channel: c, ScpStream: &ScpStream{Writer: c, Reader: bufio.NewReader(c)}}

	id, err := database.Add(s)
	if err != nil {
		return nil, err
	}
	s.ID = id

	// say hello to our customer
	url := fmt.Sprintf("%s%s", viper.GetString("ADVERTISE_URL"), path.Join("sink", s.ID))
	fmt.Fprintf(c.Stderr(), SinkBanner, url, url, url)

	return s, nil
}

// Packer fullfills database.Transfer by providing an error message
func (s *Sink) Packer() (packer.PackerCloser, error) {
	return nil, fmt.Errorf("%T cannot accept files", s)
}

// PackTo accepts a PackerCloser and adds files from the transfer to it
func (s *Sink) PackTo(p packer.PackerCloser) error {

	err := s.Pack(p)
	if err != nil && err != io.EOF {
		log.Printf("Sink error: %s", err)

		// indicate to the remote scp client we have failed
		_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 1}))

		// close stuff
		_ = s.channel.Close()
		_ = p.Close()
		return err

	}

	err = p.Close()
	if err != nil {
		log.Printf("Sink error: could not close zip file: %s", err)

		// indicate to the remote scp client we have failed
		_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 1}))

		// close stuff
		_ = s.channel.Close()
		_ = p.Close()
		return err
	}

	// indicate to remote scp client we have succeded
	_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 0}))
	_ = s.channel.Close()

	// its really not true zero bytes where written
	return nil
}
