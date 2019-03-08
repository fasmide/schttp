package scp

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
)

type ScpStream struct {
	io.Writer
	*bufio.Reader
}
type Item struct {
	io.Reader
	Name      string
	Mode      os.FileMode
	Length    int64
	Directory bool
}

type ScpCommand struct {
	raw    []byte
	fields []string
}

func NewScpCommand(input []byte) *ScpCommand {
	return &ScpCommand{raw: input, fields: strings.Fields(string(input))}
}

func (s *ScpCommand) File() bool {
	if s.raw[0] == 'C' {
		return true
	}
	return false
}

func (s *ScpCommand) Directory() bool {
	if s.raw[0] == 'D' {
		return true
	}
	return false
}

func (s *ScpCommand) Name() string {
	return strings.Trim(s.fields[2], "\n\r\x0A")
}

func (s *ScpCommand) Mode() os.FileMode {
	i64, err := strconv.ParseUint(string(s.raw[1:4]), 10, 32)
	if err != nil {
		log.Printf("ScpCommand: unable to parse file mode from %s: %s", string(s.raw[1:4]), err)
		return os.FileMode(0667)
	}
	return os.FileMode(uint32(i64))
}

func (s ScpCommand) Length() int64 {
	l, err := strconv.ParseInt(s.fields[1], 10, 64)
	if err != nil {
		log.Printf("ScpCommand: unable to parse file length: %s", err)
	}
	return l
}

func (s *ScpStream) Next() (*Item, error) {
	// ask scp client to advance
	s.Write([]byte{0x00})

	// an scp command looks something like this
	//   C0664 352 test-node-ssl-js<0x0A || LineFeed>
	raw, err := s.ReadBytes(byte(0x0A))
	if err != nil && err != io.EOF {
		return nil, fmt.Errorf("ScpScream: unable to find next scp command: %s", err)
	}

	scpCommand := NewScpCommand(raw)

	// Other scp commands exists - such as methods to set access times and modified times
	// but we dont care about these for now
	if !scpCommand.File() && !scpCommand.Directory() {
		log.Printf("Not a file or directory:", scpCommand.File(), scpCommand.Directory())
		return s.Next()
	}

	i := Item{
		Directory: scpCommand.Directory(),
		Name:      scpCommand.Name(),
		Mode:      scpCommand.Mode(),
		Length:    scpCommand.Length(),
	}
	i.Reader = io.LimitReader(s, i.Length)

	// ask scp client to advance
	s.Write([]byte{0x00})
	return &i, nil
}
