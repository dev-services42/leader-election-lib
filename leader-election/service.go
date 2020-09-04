package election

import (
	"context"
	"github.com/dev-services42/leader-election-lib/leader-election/keys"
	"github.com/dev-services42/leader-election-lib/leader-election/sessions"
	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
	"time"
)

type Service struct {
	sessions    *sessions.Service
	keys        *keys.Service
	logger      *zap.Logger
	ttl         time.Duration
	sessionName string
	keyName     string
	consul      *api.Client
}

func New(logger *zap.Logger, consul *api.Client, srvSessions *sessions.Service, srvKeys *keys.Service, ttl time.Duration, sessionName, keyName string) (*Service, error) {
	return &Service{
		logger:      logger,
		consul:      consul,
		sessions:    srvSessions,
		keys:        srvKeys,
		ttl:         ttl,
		sessionName: sessionName,
		keyName:     keyName,
	}, nil
}

func (s *Service) RunLeaderElection(ctx context.Context) <-chan bool {
	ch := make(chan bool)

	go func() {
		err := s.sessions.CreateRenew(ctx, s.ttl, s.sessionName)
		if err != nil {
			s.logger.Fatal("cannot create/renew session", zap.Error(err))
		}
	}()

	keysChanged := s.keys.WatchKeyChanges(ctx, s.keyName)

	go func() {
		defer close(ch)

		var key *api.KVPair
		var sessionID string
		for {
			select {
			case <-ctx.Done():
				return
			case sid, ok := <-s.sessions.GetSessionID():
				if !ok {
					return
				}

				sessionID = sid
			case k, ok := <-keysChanged:
				if !ok {
					return
				}

				key = k
			}

			master := s.handleChanges(ctx, sessionID, key)
			select {
			case <-ctx.Done():
			case ch <- master:
			}
		}
	}()

	return ch
}

func (s *Service) handleChanges(ctx context.Context, sessionID string, key *api.KVPair) bool {
	if sessionID == "" {
		// нужно подождать создания сессии
		return false
	}

	nodeName, err := s.consul.Agent().NodeName()
	if err != nil {
		return false
	}

	if key == nil || key.Session == "" {
		key = &api.KVPair{
			Key:     s.keyName,
			Value:   []byte(nodeName),
			Session: sessionID,
		}
		opts := (&api.WriteOptions{}).WithContext(ctx)
		acquire, _, err := s.consul.KV().Acquire(key, opts)
		if err != nil {
			return false
		}

		return acquire
	}

	if string(key.Value) != nodeName {
		return false
	}

	if key.Session != sessionID {
		return false
	}

	return true
}
