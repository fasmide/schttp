package scp

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"strings"
	"sync"

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
	
    Read more at https://github.com/fasmide/schttp

    Usage:
        scp -r someDirectory/ scp.click:

    You will then be presented with a one time URL.

`

type Server struct {
	sync.Mutex

	listener  net.Listener
	sshConfig *ssh.ServerConfig

	// this bool indicates if we have been shutdown
	// - when shutdown the server should not accept any
	//   more sinks or sources
	// - its kind of hacky but it allows us to turn down clients which was just accepted
	//   but say - was slow receiving the ssh banner - without having to track every single
	//   net.Conn and their state (are they currently tranfering data?) - and disconnect
	//   only those which are not
	// - once a transfer have started - its up to the http server to end the session
	shutdown        bool
	shutdownMessage string
}

func NewServer() *Server {
	// ssh.ServerConfig
	// - Anyone can login with any combination of user and password
	// - Any public key is accepted
	config := &ssh.ServerConfig{
		NoClientAuth:   true,
		ServerVersion:  "SSH-2.0-schttp",
		BannerCallback: SSHBanner,
		Config: ssh.Config{
			// Add in the default preferred ciphers minus chacha20 Poly
			// as we would like AES-NI acceleration
			Ciphers: []string{
				"aes128-gcm@openssh.com",
				"aes128-ctr", "aes192-ctr", "aes256-ctr",
			},
		},
	}

	// Read private key
	privateBytes, err := ioutil.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}

	hostkey, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}

	config.AddHostKey(hostkey)

	return &Server{
		sshConfig: config,
	}
}

func SSHBanner(meta ssh.ConnMetadata) string {
	return fmt.Sprintf(Banner, meta.RemoteAddr().String())
}

// Shutdown sends a message to all clients with transfers that have yet to start
// and disconnects them
func (s *Server) Shutdown(msg string) {
	// we should not accept any more connections
	s.listener.Close()

	s.Lock()

	// set shutdown bit + message
	s.shutdown = true

	// we are racing with clients that are currently authenticating
	// save the massage and turn them down when they are ready
	s.shutdownMessage = msg

	s.Unlock()
}

// Listen listens for new ssh connections
func (s *Server) Listen(listener net.Listener) {
	// set our own listener - this listener is closed later when
	// or if the server is shutdown
	s.listener = listener

	for {
		nConn, err := listener.Accept()
		if err != nil {
			// this could be normal - ie when doing upgrades
			log.Printf("unable to accept incoming ssh connection: %s", err)
			break
		}
		go s.acceptSCP(nConn)
	}
}

func (s *Server) acceptSCP(c net.Conn) {
	_, chans, reqs, err := ssh.NewServerConn(c, s.sshConfig)

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
					fmt.Fprint(channel.Stderr(), "    You seem to have specified -p (preserve create and modified time) - this is ignored\n")
				}

				// source (accept files)
				if strings.Index(payload, "-t") >= 0 {
					source, err := NewSource(channel)
					if err != nil {
						log.Printf("could not create new source: %s", err)

						// tell remote to go away
						req.Reply(false, nil)
						continue
					}

					log.Printf("source from %s, with id %s", c.RemoteAddr().String(), source.ID)

					s.Lock()
					// turn down request if we have been shutdown
					if s.shutdown {
						s.Unlock()
						fmt.Fprint(channel.Stderr(), s.shutdownMessage)
						req.Reply(false, nil)
						continue
					}
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
