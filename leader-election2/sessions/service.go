package sessions

import (
	"context"
	"github.com/cenkalti/backoff/v4"
	"github.com/hashicorp/consul/api"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type Service struct {
	consul *api.Client
	logger *zap.Logger

	sessionID   string
	sessionIDMu *sync.RWMutex
}

func New(logger *zap.Logger, consul *api.Client) (*Service, error) {
	return &Service{
		consul: consul,
		logger: logger,

		sessionIDMu: new(sync.RWMutex),
	}, nil
}

func (s *Service) CreateRenew(ctx context.Context, ttl time.Duration, sessionName string) error {
	nodeName, err := s.consul.Agent().NodeName()
	if err != nil {
		return errors.Wrap(err, "cannot get agent node name")
	}

	fn := func() error {
		s.sessionIDMu.Lock()
		s.sessionID = ""
		s.sessionIDMu.Unlock()

		doneCh := make(chan struct{})
		defer close(doneCh)

		for {
			var sessionID string
			session, err := s.getSessionByName(ctx, nodeName, sessionName)
			if err != nil {
				opts := new(api.WriteOptions).WithContext(ctx)
				sessionID, _, err = s.consul.Session().Create(&api.SessionEntry{
					Name:     sessionName,
					Behavior: "release",
					TTL:      ttl.String(),
				}, opts)
				if err != nil {
					return errors.Wrap(err, "cannot create session")
				}
			} else {
				sessionID = session.ID
			}

			s.sessionIDMu.Lock()
			s.sessionID = sessionID
			s.sessionIDMu.Unlock()

			opts := new(api.WriteOptions).WithContext(ctx)
			err2 := s.consul.Session().RenewPeriodic(ttl.String(), sessionID, opts, doneCh)
			if err2 != nil {
				return errors.Wrap(err2, "cannot renew session")
			}
		}
	}

	if err := backoff.Retry(fn, backoff.NewExponentialBackOff()); err != nil {
		// Handle error.
		return errors.Wrap(err, "backoff error")
	}

	return nil
}

func (s *Service) getSessionByName(ctx context.Context, nodeName, sessionName string) (*api.SessionEntry, error) {
	opts := (&api.QueryOptions{}).WithContext(ctx)
	sessions, _, err := s.consul.Session().Node(nodeName, opts)
	if err != nil {
		return nil, errors.Wrap(err, "cannot get sessions from consul")
	}

	for _, session := range sessions {
		if session.Name != sessionName {
			continue
		}

		return session, nil
	}

	return nil, errors.Wrap(ErrNotFound, "session not found")
}

func (s *Service) GetSessionID() (string, ) {
	s.sessionIDMu.RLock()
	defer s.sessionIDMu.RUnlock()

	return s.sessionID
}
