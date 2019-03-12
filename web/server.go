package web

import (
	"io"
	"log"
	"net"
	"net/http"
	"path"
	"strings"

	"github.com/spf13/viper"
)

func init() {
	viper.SetDefault("HTTP_LISTEN", "0.0.0.0:8080")
	viper.SetDefault("ADVERTISE_URL", "http://localhost:8080/")

}

type Server struct {
	http.ServeMux

	// We will be looking up sinks and sources from the database
	// of connected sinks and sources
	DB DB
}

// DB specifies methods to find sinks and sources
// - these must be thread safe
type DB interface {
	Sink(string) (io.WriterTo, error)
	Source(string) (io.ReaderFrom, error)
}

func (s *Server) Listen() {
	// setup routes
	s.HandleFunc("/sink/", s.Sink)
	s.HandleFunc("/source/", s.Source)

	// Setup listener
	l, err := net.Listen("tcp", viper.GetString("HTTP_LISTEN"))
	if err != nil {
		log.Fatalf("HTTP: unable to listen on %s: %s", l.Addr().String(), err)
	}
	log.Printf("HTTP: listening on %s", l.Addr().String())

	// Listen for http
	http.Serve(l, s)
}

func (s *Server) Sink(w http.ResponseWriter, r *http.Request) {
	id := path.Base(r.URL.Path)

	// remove .zip if pressent
	id = strings.Replace(id, ".zip", "", 1)

	sink, err := s.DB.Sink(id)
	if err != nil {
		// the only error available from Sink is a 404 style error
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	log.Printf("%s sinks %s", r.RemoteAddr, r.URL.Path)

	_, err = sink.WriteTo(w)
	if err != nil {
		log.Printf("HTTP: failed to sink data to %s: %s", r.RemoteAddr, err)
	}

}

func (s *Server) Source(w http.ResponseWriter, r *http.Request) {

}
