package scp

import (
	"io"
	"io/ioutil"
	"log"
	"os"
)

type TestPacker struct {
}

func (t *TestPacker) File(name string, mode os.FileMode, r io.Reader) error {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("TestError: %s", err)
	}
	log.Printf("Test: read file %s, with content %s", name, d)

	return nil
}

func (t *TestPacker) Enter(name string, mode os.FileMode) error {
	log.Printf("Test: Creating directory %s with mode %s", name, mode)
	return nil
}

func (t *TestPacker) Exit() error {
	log.Printf("Test: Leaving")
	return nil
}
