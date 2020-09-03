package main

import (
	"context"
	"fmt"
	"github.com/dev-services42/leader-election/leader-election2/sessions"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"time"
)

const (
	ttl         = 10 * time.Second
	sessionName = "services/my-service/leader"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}

	config := api.DefaultConfig() // Create a new api client config
	config.Address = "127.0.0.1:8500"
	consul, err := api.NewClient(config) // Create a Consul api client
	if err != nil {
		panic(err)
	}

	sess, err := sessions.New(logger, consul)
	if err != nil {
		panic(err)
	}

	go func() {
		err := sess.CreateRenew(ctx, ttl, sessionName)
		if err != nil {
			logger.Fatal("cannot create/renew session", zap.Error(err))
		}
	}()

	for {
		fmt.Println(sess.GetSessionID())
		time.Sleep(time.Second)
	}

}
