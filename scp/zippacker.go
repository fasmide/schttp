package scp

import (
	"archive/zip"
	"io"
	"io/ioutil"
	"log"
	"os"
)

type ZipPacker struct {
	zip.Writer
	Path string
}

func (z *ZipPacker) File(name string, mode os.FileMode, r io.Reader) error {
	d, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("ZipPacker: %s", err)
	}
	log.Printf("ZipPacker: read file %s, with content %s", name, d)

	return nil
}

func (z *ZipPacker) Enter(name string, mode os.FileMode) error {
	log.Printf("ZipPacker: Creating directory %s with mode %s", name, mode)
	return nil
}

func (z *ZipPacker) Exit() error {
	log.Printf("ZipPacker: Leaving")
	return nil
}
