package main

import (
	client "client/internal/app"
	"envconfig"
	"log"
	"net/url"

	"go.uber.org/zap"
)

func main() {
	lg, e := zap.NewProduction()
	if e != nil {
		log.Fatal("Failed to init logger")
	}

	es := envconfig.NewEnvClientStorage()

	u := url.URL{Scheme: "ws", Host: es.EnvGetAddr("serverOutsideAddr"), Path: "/"}
	if e = client.RunClient(u.String(), lg); e != nil {
		if e == client.ErrorSigQuit {
			lg.Info("Client stopped running", zap.Error(e))
		} else {
			lg.Fatal("Client stopped running unexpectedly", zap.Error(e))
		}
		return
	}
	lg.Info("Client stopped running")
}
