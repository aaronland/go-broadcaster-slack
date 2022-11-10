package encode

import (
	"context"
	"image"
	"io/ioutil"
	"os"
)

func testEncoder(uri string) error {

	ctx := context.Background()

	fh, err := os.Open("fixtures/tokyo.jpg")

	if err != nil {
		return err
	}

	defer fh.Close()

	im, _, err := image.Decode(fh)

	if err != nil {
		return err
	}

	enc, err := NewEncoder(ctx, uri)

	if err != nil {
		return err
	}

	devnull := ioutil.Discard
	return enc.Encode(ctx, im, devnull)
}
