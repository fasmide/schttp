package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/cloudflare/tableflip"
	"github.com/fasmide/schttp/web"
	"github.com/spf13/viper"

	"github.com/fasmide/schttp/scp"
)

func init() {
	viper.SetDefault("HTTP_LISTEN", "0.0.0.0:8080")
	viper.SetDefault("SSH_LISTEN", "0.0.0.0:2222")
	viper.SetDefault("PID_FILE", "/var/run/schttp.pid")
}

// main purpose is to set listeners up, handle process replacement (upgrades) and shut things down nicely
func main() {
	viper.AutomaticEnv()

	// detect if we are being run by systemd (or whatever)
	if os.Getppid() == 1 {
		// in systemd - remove the timestamps as journald adds this it self
		// - without the timestamp its also possible for journald to detect identical messages
		// - the following magic was found on stackoverflow :)
		log.SetFlags(log.Flags() &^ (log.Ldate | log.Ltime))
	} else {
		// when not run by systemd
		// - add PID to easier allow debugging of upgrades
		// - journald adds the pid of the process just as it adds timestamps
		log.SetPrefix(fmt.Sprintf("[%d] ", os.Getpid()))
	}

	s, err := NewSchttp()
	if err != nil {
		log.Fatalf("could not initialize schttp: %s", err)
	}

	err = s.Run()
	if err != nil {
		log.Fatalf("schttp run failed: %s", err)
	}

	s.upgrader.Stop()
	log.Printf("Exited normally")
}

type schttp struct {
	upgrader *tableflip.Upgrader

	httpFd net.Listener
	sshFd  net.Listener

	webServer *web.Server
	scpServer *scp.Server
}

// NewSchttp returns a new schttp which represents the schttp as a whole
// and handles gracefull upgrades
func NewSchttp() (*schttp, error) {

	// init tableflip upgrader
	upgrader, err := tableflip.New(tableflip.Options{
		PIDFile: viper.GetString("PID_FILE"),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to initiate tableflip: %s", err)
	}
	s := schttp{
		upgrader: upgrader,
	}

	// handle HUP signals in its own routine
	go s.HandleSIGHUP()

	// setup ssh listener - tableflip will create a new or inherit the old schttp's listener
	listener, err := upgrader.Fds.Listen("tcp", viper.GetString("SSH_LISTEN"))
	if err != nil {
		return nil, fmt.Errorf("SSH: unable to listen on %s: %s", viper.GetString("SSH_LISTEN"), err)
	}

	log.Printf("SSH: listening on %s", listener.Addr().String())
	s.sshFd = listener

	// setup http listener
	listener, err = upgrader.Fds.Listen("tcp", viper.GetString("HTTP_LISTEN"))
	if err != nil {
		return nil, fmt.Errorf("HTTP: unable to listen on %s: %s", viper.GetString("HTTP_LISTEN"), err)
	}

	log.Printf("HTTP: listening on %s", listener.Addr().String())
	s.httpFd = listener

	return &s, nil
}

func (s *schttp) HandleSIGHUP() {
	// Do an upgrade on SIGHUP
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP)
	for range sig {
		err := s.upgrader.Upgrade()
		if err != nil {
			log.Println("Upgrade failed:", err)
		}

		// we should not stop handling HUPs here- there may be future upgrade tries
	}
}

// Run should return only when both servers have been stopped or have crashed
func (s *schttp) Run() error {
	s.scpServer = scp.NewServer()
	go s.scpServer.Listen(s.sshFd)

	s.webServer = &web.Server{DB: s.scpServer}
	go s.webServer.Listen(s.httpFd)

	log.Printf("schttp is alive")

	// indicate to the upgrader that we are ready
	if err := s.upgrader.Ready(); err != nil {
		return err
	}

	// wait here until the upgrader tells us to exit
	<-s.upgrader.Exit()

	log.Printf("Shutting down for upgrade")
	// TODO: Make sure to set a deadline on exiting the process
	// after upg.Exit() is closed. No new upgrades can be
	// performed if the parent doesn't exit.
	// time.AfterFunc(30*time.Second, func() {
	// 	log.Println("Graceful shutdown timed out")
	// 	os.Exit(1)
	// })

	s.scpServer.Shutdown("\n    Software upgrade - please reconnect\n\n")

	// Wait for connections to drain.
	return s.webServer.Shutdown(context.Background())
}
