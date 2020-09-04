package main

import (
	"context"
	"fmt"
	"github.com/dev-services42/leader-election-lib/leader-election"
	"github.com/dev-services42/leader-election-lib/leader-election/keys"
	"github.com/dev-services42/leader-election-lib/leader-election/sessions"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"os"
	"time"
)

const (
	ttl         = 30 * time.Second
	sessionName = "services/my-service/leader"
	keyName     = sessionName
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	config := api.DefaultConfig() // Create a new api client config
	config.Address = os.Getenv("CONSUL_ADDR")
	consul, err := api.NewClient(config) // Create a Consul api client
	if err != nil {
		panic(err)
	}

	sess, err := sessions.New(logger, consul)
	if err != nil {
		panic(err)
	}

	sKeys, err := keys.New(logger, consul, 10*time.Second)
	if err != nil {
		panic(err)
	}

	srv, err := election.New(
		logger,
		consul,
		sess,
		sKeys,
		ttl,
		sessionName,
		keyName,
	)
	if err != nil {
		panic(err)
	}

	masterCh := srv.RunLeaderElection(ctx)
	for master := range masterCh {
		fmt.Println(master)
	}

	<-make(chan int)
}
