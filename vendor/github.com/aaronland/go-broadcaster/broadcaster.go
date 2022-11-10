package broadcaster

import (
	"context"
	"fmt"
	"github.com/aaronland/go-roster"
	"github.com/aaronland/go-uid"
	"image"
	"log"
	"net/url"
	"sort"
	"strings"
)

// Broadcaster provides a minimalist interface for "broadcasting" messages to an arbitrary service or target.
type Broadcaster interface {
	// BroadcastMessage "broadcasts" a `Message` struct.
	BroadcastMessage(context.Context, *Message) (uid.UID, error)
	// SetLogger assigns a specific `log.Logger` instance to be used for logging messages.
	SetLogger(context.Context, *log.Logger) error
}

// Message defines a struct containing properties to "broadcast". The semantics of these
// properties are determined by the server or target -specific implementation of the `Broadcaster`
// interface.
type Message struct {
	// Title is a string to use as the title of a "broadcast" message.
	Title string
	// Body is a string to use as the body of a "broadcast" message.
	Body string
	// Images is zero or more `image.Image` instances to be included with a "broadcast" messages.
	// Images are encoded according to rules implemented by service or target -specific implementation
	// of the `Broadcaster` interface.
	Images []image.Image
}

var broadcaster_roster roster.Roster

// BroadcasterInitializationFunc is a function defined by individual broadcaster package and used to create
// an instance of that broadcaster
type BroadcasterInitializationFunc func(ctx context.Context, uri string) (Broadcaster, error)

// RegisterBroadcaster registers 'scheme' as a key pointing to 'init_func' in an internal lookup table
// used to create new `Broadcaster` instances by the `NewBroadcaster` method.
func RegisterBroadcaster(ctx context.Context, scheme string, init_func BroadcasterInitializationFunc) error {

	err := ensureBroadcasterRoster()

	if err != nil {
		return err
	}

	return broadcaster_roster.Register(ctx, scheme, init_func)
}

func ensureBroadcasterRoster() error {

	if broadcaster_roster == nil {

		r, err := roster.NewDefaultRoster()

		if err != nil {
			return err
		}

		broadcaster_roster = r
	}

	return nil
}

// NewBroadcaster returns a new `Broadcaster` instance configured by 'uri'. The value of 'uri' is parsed
// as a `url.URL` and its scheme is used as the key for a corresponding `BroadcasterInitializationFunc`
// function used to instantiate the new `Broadcaster`. It is assumed that the scheme (and initialization
// function) have been registered by the `RegisterBroadcaster` method.
func NewBroadcaster(ctx context.Context, uri string) (Broadcaster, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	scheme := u.Scheme

	i, err := broadcaster_roster.Driver(ctx, scheme)

	if err != nil {
		return nil, err
	}

	init_func := i.(BroadcasterInitializationFunc)
	return init_func(ctx, uri)
}

// Schemes returns the list of schemes that have been registered.
func Schemes() []string {

	ctx := context.Background()
	schemes := []string{}

	err := ensureBroadcasterRoster()

	if err != nil {
		return schemes
	}

	for _, dr := range broadcaster_roster.Drivers(ctx) {
		scheme := fmt.Sprintf("%s://", strings.ToLower(dr))
		schemes = append(schemes, scheme)
	}

	sort.Strings(schemes)
	return schemes
}
