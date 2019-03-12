package scp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"path"

	"github.com/rs/xid"
	"github.com/spf13/viper"
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
	c.Stderr().Write([]byte(fmt.Sprintf("[scp.click] Download from %s%s\n", viper.GetString("ADVERTISE-URL"), path.Join("sink", s.ID.String()))))

	return s
}

// WriteTo implements the default golang WriterTo interface
// It will read the files from the remote client and pack them up in zip format
func (s *Sink) WriteTo(w io.Writer) (int64, error) {
	z := NewZipPacker(w)

	err := s.Pack(z)
	if err != nil && err != io.EOF {
		log.Printf("Sink error: %s", err)

		// indicate to the remote scp client we have failed
		_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 1}))

		// close stuff
		_ = s.channel.Close()
		_ = z.Close()
		return 0, err

	}

	err = z.Close()
	if err != nil {
		log.Printf("Sink error: could not close zip file: %s", err)
		// indicate to the remote scp client we have failed
		_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 1}))

		// close stuff
		_ = s.channel.Close()
		_ = z.Close()
	}

	// indicate to remote scp client we have succeded
	_, _ = s.channel.SendRequest("exit-status", false, ssh.Marshal(&ExitStatus{Status: 0}))
	_ = s.channel.Close()

	// its really not true zero bytes where written
	return 0, nil
}
