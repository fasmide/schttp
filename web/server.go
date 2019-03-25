package web

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/fasmide/schttp/packer"
	"github.com/gobuffalo/packr/v2"
	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("ADVERTISE_URL", "http://localhost:8080/")
}

type Server struct {
	http.ServeMux
	http.Server

	// We will be looking up sinks and sources from the database
	// of connected sinks and sources
	DB DB

	box *packr.Box
}

// DB specifies methods to find sinks and sources
// - these must be thread safe
type DB interface {
	Sink(string) (packer.PackerTo, error)
	Source(string) (io.ReaderFrom, error)
}

func (s *Server) Listen(l net.Listener) {
	// set up a new box by giving it a name and an optional (relative) path to a folder on disk:
	s.box = packr.New("static", "../static")

	// setup routes
	s.HandleFunc("/sink/", s.Sink)
	s.HandleFunc("/source/", s.Source)
	s.Handle("/static/", http.FileServer(s.box))

	// this is kind of a hack but im unable to make the packr.Box serve the index.html by it self
	// this will however make any other requests receive the index.html which is properly ok
	s.HandleFunc("/", s.Index)

	// the handler is embedded in s
	s.Server.Handler = s

	// Listen for http
	s.Serve(l)
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	b, err := s.box.Find("index.html")
	if err != nil {
		http.Error(w, "im somehow without an index.html file", http.StatusNotFound)
		return
	}
	w.Write(b)
}

func (s *Server) Sink(w http.ResponseWriter, r *http.Request) {
	// figure out id and file extension
	fileParts := strings.SplitN(path.Base(r.URL.Path), ".", 2)

	// ensure there was an file extension given
	if len(fileParts) != 2 {
		http.Error(w, "please add file extension, e.g. .zip or .tar.gz", http.StatusBadRequest)
		return
	}

	// the real id is the first part of ext
	id := fileParts[0]
	extension := fileParts[1]

	// figure out a packer to use
	var p packer.PackerCloser
	if extension == "zip" {
		p = packer.NewZip(w)
	}
	if extension == "tar.gz" {
		p = packer.NewTarGz(w)
	}

	// if the above did not result in a packer - stop
	if p == nil {
		http.Error(
			w,
			fmt.Sprintf("i cannot do \"%s\" files - please add .zip or .tar.gz only", extension),
			http.StatusBadRequest,
		)

		return
	}

	// find the sink in question
	sink, err := s.DB.Sink(id)
	if err != nil {
		// the only error available from Sink is a 404 style error
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	log.Printf("%s sinks %s", r.RemoteAddr, r.URL.Path)

	// Pack sink contents to packer
	err = sink.PackTo(p)
	if err != nil {
		log.Printf("HTTP: failed to sink data to %s: %s", r.RemoteAddr, err)
	}

}

func (s *Server) Source(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	n, _ := io.Copy(ioutil.Discard, r.Body)
	log.Printf("Just discarded %d bytes", n)
}
