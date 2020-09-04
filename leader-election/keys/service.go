package keys

import (
	"context"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"time"
)

type Service struct {
	logger   *zap.Logger
	consul   *api.Client
	waitTime time.Duration
}

func New(logger *zap.Logger, consul *api.Client, waitTime time.Duration) (*Service, error) {
	return &Service{
		logger:   logger,
		consul:   consul,
		waitTime: waitTime,
	}, nil
}

func (s *Service) WatchKeyChanges(ctx context.Context, key string) <-chan *api.KVPair {
	ch := make(chan *api.KVPair)

	go func() {
		defer close(ch)

		delay := 300 * time.Millisecond
		var waitIndex uint64

		for {
			select {
			case <-ctx.Done():
				return
			default:
				opts := (&api.QueryOptions{
					WaitIndex: waitIndex,
					WaitTime:  s.waitTime,
				}).WithContext(ctx)
				res, _, err := s.consul.KV().Get(key, opts)
				if err != nil {
					s.logger.Warn("cannot get key", zap.Error(err))
					time.Sleep(delay)
					continue
				}
				if res == nil {
					ch <- nil
					time.Sleep(delay)
					continue
				}

				ch <- res
				waitIndex = res.ModifyIndex
			}
		}
	}()

	return ch
}
