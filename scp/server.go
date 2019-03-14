package scp

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh"
)

const Banner = `
     ___  ___ _ __  
    / __|/ __| '_ \ 
    \__ \ (__| |_) |
    |___/\___| .__/ 
            | |    
            |_|    
             _ _      _    
            | (_)    | |   
         ___| |_  ___| | __
        / __| | |/ __| |/ /
       | (__| | | (__|   < 
        \___|_|_|\___|_|\_\
        (click, not dick)
						
    Hello %s, you have reached scp.click.
    
    This service will enable you to transfer files between
    boxes using standard tools such as scp, curl and unzip.

    Usage:
        scp -r someDirectory/ scp.click:

    You will then be presented with a one time URL.

    Read more at https://github.com/fasmide/schttp

`

func init() {
	viper.SetDefault("SSH_LISTEN", "0.0.0.0:2222")
}

type Server struct {
	sync.Mutex

	sinks   map[string]*Sink
	sources map[string]*Source
}

func NewServer() *Server {
	return &Server{sinks: make(map[string]*Sink), sources: make(map[string]*Source)}
}

func (s *Server) Banner(meta ssh.ConnMetadata) string {
	return fmt.Sprintf(Banner, meta.RemoteAddr().String())
}

func (s *Server) Sink(id string) (io.WriterTo, error) {
	s.Lock()
	defer s.Unlock()

	if sink, exists := s.sinks[id]; exists {
		delete(s.sinks, id)
		return sink, nil
	}
	return nil, fmt.Errorf("%s does not exist", id)
}

func (s *Server) Source(id string) (io.ReaderFrom, error) {
	s.Lock()
	defer s.Unlock()

	if source, exists := s.sources[id]; exists {
		delete(s.sources, id)
		return source, nil
	}
	return nil, fmt.Errorf("%s does not exist", id)
}

// Listen listens for new ssh connections
func (s *Server) Listen() {

	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	hostkey, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	// anyone can login with any combination of user / password
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return nil, nil
		},

		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			return nil, nil
		},
		ServerVersion:  "SSH-2.0-scp.click",
		BannerCallback: s.Banner,
	}

	config.AddHostKey(hostkey)

	listener, err := net.Listen("tcp", viper.GetString("SSH_LISTEN"))

	log.Printf("SSH: listening on %s", listener.Addr().String())
	if err != nil {
		log.Fatal("failed to listen for ssh connections: ", err)
	}

	for {
		nConn, err := listener.Accept()
		if err != nil {
			log.Fatal("unable to accept incoming ssh connection: ", err)
			continue
		}
		go s.acceptSCP(nConn, config)
	}
}

func (s *Server) acceptSCP(c net.Conn, sshc *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(c, sshc)

	if err != nil {
		log.Printf("unable to accept ssh from %s: %s", c.RemoteAddr().String(), err)
	}

	// The incoming Request channel must be serviced - but we dont care about them
	go ssh.DiscardRequests(reqs)

	// Service the incoming Channel channel.
	for newChannel := range chans {
		// Channels have a type, depending on the application level
		// protocol intended. In the case of a shell, the type is
		// "session" and ServerShell may be used to present a simple
		// terminal interface.
		if newChannel.ChannelType() != "session" {
			log.Printf("unknown channel type %s", newChannel.ChannelType())
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			log.Printf("Could not accept channel: %v", err)
			continue
		}

		// Sessions have out-of-band requests such as "shell",
		// "pty-req" and "env".  Here we handle only the
		// "shell" request.
		go func(in <-chan *ssh.Request) {
			for req := range in {
				// exec with payload scp -t || -f is allowed
				if req.Type != "exec" {
					req.Reply(false, nil)
					continue
				}

				// so - the first 4 bytes are ... i dont know ...
				payload := string(req.Payload[4:])

				// does the command start with scp ?
				if !strings.HasPrefix(payload, "scp") {
					req.Reply(false, nil)
					continue
				}

				// if the user specified "-p" tell him it wont do anything
				if strings.Index(payload, "-p") >= 0 {
					fmt.Fprint(channel.Stderr(), "[scp.click] You seem to have specified -p (preserve create and modified time) - this is ignored\n")
				}

				// sink (accept files)
				if strings.Index(payload, "-t") >= 0 {
					sink, err := NewSink(channel)
					if err != nil {
						log.Printf("could not create new sink: %s", err)

						// tell remote to go away
						req.Reply(false, nil)
						channel.Close()
						continue
					}

					log.Printf("Sink from %s, with id %s", c.RemoteAddr().String(), sink.ID)

					s.Lock()
					s.sinks[sink.ID] = sink
					s.Unlock()

					req.Reply(true, nil)
					continue
				}

				// source (send files)
				if strings.Index(payload, "-f") >= 0 {

					fmt.Fprintf(channel.Stderr(), "Sourcing is not supported ... yet :)")
					req.Reply(false, nil)
					continue
				}

				// default
				log.Printf("unable to handle scp requests without -t or -f: \"%s\"", payload)
				req.Reply(false, nil)
			}
		}(requests)
	}
}
