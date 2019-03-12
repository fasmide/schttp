package main

import (
	"github.com/fasmide/schttp/web"
	"github.com/spf13/viper"

	"github.com/fasmide/schttp/scp"
)

func main() {
	viper.AutomaticEnv()

	scp := scp.NewServer()
	go scp.Listen()

	web := web.Server{DB: scp}
	web.Listen()
}
