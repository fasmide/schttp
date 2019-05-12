package scp

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fasmide/schttp/packer"
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
	TimeCreatedModified
	Exit
)

type Command struct {
	Name   string
	Mode   os.FileMode
	Length int64
	Type   Type
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
	if raw[0] == 'E' {
		c.Type = Exit
		c.Name = ""
		c.Mode = 0
		c.Length = 0
		return nil
	}
	if raw[0] == 'T' {
		// we dont fully support T
		// but we also dont need it, its about access and modified time
		c.Type = TimeCreatedModified
		c.Name = ""
		c.Mode = 0
		c.Length = 0
		return nil
	}

	if c.Type == Unsupported {
		return fmt.Errorf("unsupported scp command: \"%s\" %x", string(raw), raw)
	}

	i64, err := strconv.ParseUint(string(raw[1:4]), 8, 32)
	if err != nil {
		return fmt.Errorf("unable to parse file mode from %s: %s", string(raw[1:4]), err)
	}
	c.Mode = os.FileMode(uint32(i64))

	// split by space into fields for Name and Length
	fields := strings.Fields(string(raw))

	// Name is the third field and beyond
	// TODO: dont use Fields - use some kind of ReadUntil or something
	c.Name = strings.Trim(strings.Join(fields[2:], " "), "\n\r\x0A")

	l, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil {
		return fmt.Errorf("unable to parse file length: %s", err)
	}
	c.Length = l

	return nil
}

// Pack reads files from an scp client and packs them with a given Packer
func (s *ScpStream) Pack(p packer.Packer) error {
	// until something returns...
	for {

		// ask remote client to advance
		_, err := s.Write([]byte{0x00})
		if err != nil {
			return fmt.Errorf("unable to advance remote scp client: %s", err)
		}
		// an scp command looks something like this
		//   C0664 352 test-node-ssl-js<0x0A || LineFeed>
		var c Command
		line, err := s.ReadBytes(byte(0x0A))

		// we are finished
		if err == io.EOF {
			return err
		}

		if err != nil {
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

			// Pack the file
			err = p.File(c.Name, c.Mode, c.Length, io.LimitReader(s, c.Length))

			if err != nil {
				return fmt.Errorf("unable to pack: %s", err)
			}

			// the client will send a NUL after sending a file
			b, err := s.ReadByte()
			if err != nil {
				return fmt.Errorf("unable to read advance NUL byte: %s", err)
			}
			if b != 0x00 {
				return fmt.Errorf("advance NUL byte was not NUL: it was %x", b)
			}

		case Directory:
			p.Enter(c.Name, c.Mode)
		case Exit:
			p.Leave()
		}
	}

}
