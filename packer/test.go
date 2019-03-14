package packer

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

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

func (t *Test) Exit() error {
	log.Printf("Test: Leaving")
	return nil
}
