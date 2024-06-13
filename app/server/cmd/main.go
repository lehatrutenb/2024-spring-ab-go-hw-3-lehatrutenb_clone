package main

import (
	"envconfig"
	"server/internal/ports/websocketport"
)

func main() {
	es := envconfig.NewEnvStorage()

	websocketport.RunServer(es.EnvGetAddr("serverAddr"), map[string]string{
		"kafkaAddr":   es.EnvGetAddr("kafkaAddr"),
		"storageAddr": es.EnvGetAddr("storageServerAddr"),
	})
}
