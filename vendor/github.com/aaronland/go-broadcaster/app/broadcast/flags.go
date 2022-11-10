package broadcast

import (
	"flag"
	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
)

// One or more aaronland/go-broadcast URIs.
var broadcaster_uris multi.MultiCSVString

// The title of the message to broadcast.
var title string

// The body of the message to broadcast.
var body string

// Zero or more paths to images to include with the message to broadcast.
var image_paths multi.MultiString

func DefaultFlagSet() *flag.FlagSet {

	fs := flagset.NewFlagSet("broadcast")

	fs.Var(&broadcaster_uris, "broadcaster", "One or more aaronland/go-broadcast URIs.")

	fs.StringVar(&title, "title", "", "The title of the message to broadcast.")
	fs.StringVar(&body, "body", "", "The body of the message to broadcast.")

	fs.Var(&image_paths, "image", "Zero or more paths to images to include with the message to broadcast.")

	return fs
}
