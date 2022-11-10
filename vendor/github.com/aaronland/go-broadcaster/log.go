package broadcaster

import (
	"context"
	"github.com/aaronland/go-uid"
	"log"
	"time"
)

func init() {
	ctx := context.Background()
	RegisterBroadcaster(ctx, "log", NewLogBroadcaster)
}

// LogBroadcaster implements the `Broadcaster` interface to broadcast messages
// to a `log.Logger` instance.
type LogBroadcaster struct {
	Broadcaster
	logger *log.Logger
}

// NewLogBroadcaster returns a new `LogBroadcaster` configured by 'uri' which is expected to
// take the form of:
//
//	log://
//
// By default `LogBroadcaster` instances are configured to broadcast messages to a `log.Default`
// instance. If you want to change that call the `SetLogger` method.
func NewLogBroadcaster(ctx context.Context, uri string) (Broadcaster, error) {

	logger := log.Default()

	b := LogBroadcaster{
		logger: logger,
	}
	return &b, nil
}

// BroadcastMessage broadcast the title and body properties of 'msg' to the `log.Logger` instance
// associated with 'b'. It does not publish images yet. Maybe someday it will try to convert images
// to their ascii interpretations but today it does not. It returns the value of the Unix timestamp
// that the log message was broadcast.
func (b *LogBroadcaster) BroadcastMessage(ctx context.Context, msg *Message) (uid.UID, error) {

	b.logger.Printf("%s %s\n", msg.Title, msg.Body)

	now := time.Now()
	ts := now.Unix()

	return uid.NewInt64UID(ctx, ts)
}

// SetLoggers assigns 'logger' to 'b'.
func (b *LogBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {
	b.logger = logger
	return nil
}
