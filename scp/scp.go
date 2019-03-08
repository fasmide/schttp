package scp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

type ScpStream struct {
	io.Writer
	*bufio.Reader
}

type Type int

const (
	Unsupported Type = iota
	Create
	Directory
	Leave
)

type Command struct {
	Name   string
	Mode   os.FileMode
	Length int64
	Type   Type
}

type Packer interface {
	File(string, os.FileMode, io.Reader) error
	Enter(string, os.FileMode) error
	Leave() error1
}

func (c *Command) Parse(raw []byte) error {
	c.Type = Unsupported
	// Determinane what type of Command we are dealing with
	if raw[0] == 'C' {
		c.Type = Create
	}
	if raw[0] == 'D' {
		c.Type = Directory
	}

	if c.Type == Unsupported {
		return fmt.Errorf("unsupported scp command: %s", string(raw[0]))
	}

	i64, err := strconv.ParseUint(string(raw[1:4]), 10, 32)
	if err != nil {
		return fmt.Errorf("unable to parse file mode from %s: %s", string(raw[1:4]), err)
	}
	c.Mode = os.FileMode(uint32(i64))

	// split by space into fields for Name and Length
	fields := strings.Fields(string(raw))
	c.Name = strings.Trim(fields[2], "\n\r\x0A")

	l, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse file length: %s", err)
	}
	c.Length = l

	return nil
}

// Pack reads files from an scp client and packs them with Packer
func (s *ScpStream) Pack(p Packer) error {
	// ask remote client to advance
	_, err := s.Write([]byte{0x00})
	if err != nil {
		return fmt.Errorf("unable to advance remote scp client: %s", err)
	}

	// an scp command looks something like this
	//   C0664 352 test-node-ssl-js<0x0A || LineFeed>
	var c Command
	line, err := s.ReadBytes(byte(0x0A))
	if err != nil && err != io.EOF {
		return fmt.Errorf("unable to find next scp command: %s", err)
	}

	err = c.Parse(line)
	if err != nil {
		return fmt.Errorf("unable to parse scp command: %s", err)
	}

	switch c.Type {
	case Create:
		// ask remote client to send file
		_, err := s.Write([]byte{0x00})
		if err != nil {
			return fmt.Errorf("unable to advance remote scp client: %s", err)
		}
		p.File(c.Name, c.Mode, io.LimitReader(s, c.Length))
	case Directory:
		p.Enter(c.Name, c.Mode)
	case Leave:
		p.Leave()
	}

	return nil
}
