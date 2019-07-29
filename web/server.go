package web

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/fasmide/schttp/database"
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

	// we have our own storage of users sending files with http
	// the database does not hold transfers when they are started
	// and as http is async in nature we have to keep a reference
	// for when the next HTTP POST comes in with a file
	sources     map[string]*HTTPSource
	sourcesLock sync.RWMutex

	box *packr.Box
}

// NewServer returns a initiated http server
func NewServer() *Server {
	s := &Server{sources: make(map[string]*HTTPSource)}

	// set up a new box by giving it a name and an optional (relative) path to a folder on disk:
	s.box = packr.New("static", "../static")

	// setup routes
	s.HandleFunc("/sink/", s.Sink)
	s.HandleFunc("/source/", s.Source)
	s.HandleFunc("/newsource/", s.NewSource)
	s.Handle("/static/", http.FileServer(s.box))

	// this is kind of a hack but im unable to make the packr.Box serve the index.html by it self
	// this will however make any other requests receive the index.html which is properly ok
	s.HandleFunc("/", s.Index)

	// the handler is embedded in s
	s.Server.Handler = s

	return s
}

func (s *Server) Index(w http.ResponseWriter, r *http.Request) {
	b, err := s.box.Find("index.html")
	if err != nil {
		http.Error(w, "im somehow without an index.html file", http.StatusNotFound)
		return
	}
	w.Write(b)
}

// NewSource adds a new source and should be call'ed from the js app
func (s *Server) NewSource(w http.ResponseWriter, r *http.Request) {
	// create a new HTTPSource and add it to the database
	hs := NewHTTPSource()
	id, err := database.Add(hs)
	if err != nil {
		http.Error(w, fmt.Sprintf("cannot create new http source: %s", err), 500)
		return
	}
	hs.ID = id

	// lock sources and add this new source
	s.sourcesLock.Lock()
	s.sources[id] = hs
	s.sourcesLock.Unlock()

	dec := json.NewEncoder(w)
	err = dec.Encode(hs)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to json encode your data: %s", err), 500)
		return
	}

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
	sink, err := database.Fetch(id)
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

// Source looks up the given source and passes the http.request to HTTPSource
func (s *Server) Source(w http.ResponseWriter, r *http.Request) {
	urlParts := strings.Split(r.URL.Path, "/")

	// we know that [1] is "/source/" and [2] should be our id
	id := urlParts[2]

	s.sourcesLock.RLock()
	hs, exists := s.sources[id]
	s.sourcesLock.RUnlock()

	if !exists {
		http.Error(w, fmt.Sprintf("No httpsource with id %s exists", id), http.StatusNotFound)
		return
	}

	// we know, that everything after [0] and [1] is the path of the file including its filename

	err := hs.Accept(strings.Join(urlParts[3:], "/"), r.ContentLength, r.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("remote could not accept file: %s", err), http.StatusInternalServerError)
		return
	}
}
