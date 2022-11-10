package broadcaster

import (
	"context"
	"fmt"
	"github.com/aaronland/go-uid"
	"github.com/hashicorp/go-multierror"
	"log"
)

type MultiBroadcaster struct {
	Broadcaster
	broadcasters []Broadcaster
	logger       *log.Logger
	async        bool
}

func NewMultiBroadcasterFromURIs(ctx context.Context, broadcaster_uris ...string) (Broadcaster, error) {

	broadcasters := make([]Broadcaster, len(broadcaster_uris))

	for idx, br_uri := range broadcaster_uris {

		br, err := NewBroadcaster(ctx, br_uri)

		if err != nil {
			return nil, fmt.Errorf("Failed to create broadcaster for '%s', %v", br_uri, err)
		}

		broadcasters[idx] = br
	}

	return NewMultiBroadcaster(ctx, broadcasters...)
}

func NewMultiBroadcaster(ctx context.Context, broadcasters ...Broadcaster) (Broadcaster, error) {

	logger := log.Default()

	async := true

	b := MultiBroadcaster{
		broadcasters: broadcasters,
		logger:       logger,
		async:        async,
	}

	return &b, nil
}

func (b *MultiBroadcaster) BroadcastMessage(ctx context.Context, msg *Message) (uid.UID, error) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	th := b.newThrottle()

	done_ch := make(chan bool)
	err_ch := make(chan error)
	id_ch := make(chan uid.UID)

	for _, bc := range b.broadcasters {

		go func(bc Broadcaster, msg *Message) {

			defer func() {
				done_ch <- true
				th <- true
			}()

			<-th

			select {
			case <-ctx.Done():
				return
			default:
				// pass
			}

			id, err := bc.BroadcastMessage(ctx, msg)

			if err != nil {
				err_ch <- fmt.Errorf("[%T] Failed to broadcast message: %s\n", bc, err)
			}

			id_ch <- id

		}(bc, msg)
	}

	remaining := len(b.broadcasters)
	var result error

	ids := make([]uid.UID, 0)

	for remaining > 0 {
		select {
		case <-ctx.Done():
			return uid.NewNullUID(ctx)
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			result = multierror.Append(result, err)
		case id := <-id_ch:
			ids = append(ids, id)
		}
	}

	if result != nil {
		return nil, fmt.Errorf("One or more errors occurred, %w", result)
	}

	return uid.NewMultiUID(ctx, ids...), nil
}

func (b *MultiBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {

	b.logger = logger

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	th := b.newThrottle()

	done_ch := make(chan bool)
	err_ch := make(chan error)

	for _, bc := range b.broadcasters {

		go func(bc Broadcaster, logger *log.Logger) {

			defer func() {
				done_ch <- true
				th <- true
			}()

			<-th

			select {
			case <-ctx.Done():
				return
			default:
				// pass
			}

			err := bc.SetLogger(ctx, logger)

			if err != nil {
				err_ch <- fmt.Errorf("[%T] Failed to set logger: %v", bc, err)
			}

		}(bc, logger)
	}

	remaining := len(b.broadcasters)
	var result error

	for remaining > 0 {
		select {
		case <-ctx.Done():
			return nil
		case <-done_ch:
			remaining -= 1
		case err := <-err_ch:
			result = multierror.Append(result, err)
		}
	}

	if result != nil {
		return fmt.Errorf("One or more errors occurred, %w", result)
	}

	return nil
}

func (b *MultiBroadcaster) newThrottle() chan bool {

	workers := len(b.broadcasters)

	if !b.async {
		workers = 1
	}

	throttle := make(chan bool, workers)

	for i := 0; i < workers; i++ {
		throttle <- true
	}

	return throttle
}
