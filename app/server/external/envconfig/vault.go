package envconfig

import (
	"context"
	"log"

	vault "github.com/hashicorp/vault/api"
)

const serviceEnvAddr = "http://vault:8200"
const outsideEnvAddr = "http://localhost:8200"

type EnvStorage struct {
	c *vault.Client
}

func connectToVault(addr string) *vault.Client {
	config := vault.DefaultConfig()
	config.Address = addr
	c, err := vault.NewClient(config)
	if err != nil {
		log.Fatalf("Unable to initialize Vault client: %v", err)
	}

	c.SetToken("dev-only-token")
	return c
}

func NewEnvStorage() EnvStorage {
	return EnvStorage{connectToVault(serviceEnvAddr)}
}

func NewEnvClientStorage() EnvStorage {
	return EnvStorage{connectToVault(outsideEnvAddr)}
}

func (es EnvStorage) EnvUpdateAddr(app string, addr string) {
	if _, err := es.c.KVv2("secret").Put(context.Background(), app, map[string]interface{}{"addr": addr}); err != nil {
		log.Fatalf("Unable to write secret: %v", err)
	}
}

func (es EnvStorage) EnvGetAddr(app string) string {
	secret, err := es.c.KVv2("secret").Get(context.Background(), app)
	if err != nil {
		log.Fatalf("Unable to read secret: %v", err)
	}

	rA, ok := secret.Data["addr"].(string)
	if !ok {
		log.Fatalf("Value type assertion failed: %T %#v", secret.Data["addr"], secret.Data["addr"])
	}
	return rA
}
