package slack

import (
	"context"
	"fmt"
	"github.com/aaronland/go-broadcaster"
	"github.com/sfomuseum/go-slack"
	"github.com/aaronland/go-image-encode"
	"github.com/aaronland/go-uid"
	"github.com/sfomuseum/runtimevar"
	_ "image"
	"log"
	"net/url"
	"time"
)

func init() {
	ctx := context.Background()
	broadcaster.RegisterBroadcaster(ctx, "slack", NewSlackBroadcaster)
}

type SlackBroadcaster struct {
	broadcaster.Broadcaster
	webhook *slack.Webhook
	encoder       encode.Encoder
	logger        *log.Logger
}

func NewSlackBroadcaster(ctx context.Context, uri string) (broadcaster.Broadcaster, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	q := u.Query()

	creds_uri := q.Get("credentials")

	if creds_uri == "" {
		return nil, fmt.Errorf("Missing ?credentials= parameter")
	}

	rt_ctx, rt_cancel := context.WithTimeout(ctx, 5*time.Second)
	defer rt_cancel()

	client_uri, err := runtimevar.StringVar(rt_ctx, creds_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive URI from credentials, %w", err)
	}

	wh, err := slack.NewWebhook(ctx, client_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new Slack client, %w", err)
	}

	enc, err := encode.NewEncoder(ctx, "png://")

	if err != nil {
		return nil, fmt.Errorf("Failed to create image encoder, %w", err)
	}

	logger := log.Default()

	br := &SlackBroadcaster{
		webhook: wh,
		encoder:       enc,
		logger:        logger,
	}

	return br, nil
}

func (b *SlackBroadcaster) BroadcastMessage(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	// Images...
	
	slack_msg := &slack.Message{
		Channel: msg.Title,
		Text: msg.Body,
	}

	err := b.webhook.Post(ctx, slack_msg)

	if err != nil {
		return nil, fmt.Errorf("Failed to post message, %w", err)
	}
	
	return uid.NewStringUID(ctx, "")
}

func (b *SlackBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {
	b.logger = logger
	return nil
}
