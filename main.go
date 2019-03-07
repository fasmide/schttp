package main

import (
	"log"

	"github.com/fasmide/schttp/web"

	"github.com/fasmide/schttp/scp"
)

func main() {
	log.Printf("blarh")
	scp := scp.NewServer()
	go scp.Listen()
	web := web.Server{DB: scp}
	web.Listen()
}
