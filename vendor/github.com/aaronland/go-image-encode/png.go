package encode

import (
	"context"
	"image"
	"image/png"
	"io"
)

type PNGEncoder struct {
	Encoder
}

func init() {

	ctx := context.Background()
	err := RegisterEncoder(ctx, NewPNGEncoder, "png")

	if err != nil {
		panic(err)
	}
}

func (e *PNGEncoder) MimeType() string {
	return "image/png"
}

func (e *PNGEncoder) Extension() string {
	return ".png"
}

func NewPNGEncoder(ctx context.Context, uri string) (Encoder, error) {

	e := &PNGEncoder{}
	return e, nil
}

func (e *PNGEncoder) Encode(ctx context.Context, im image.Image, wr io.Writer) error {
	return png.Encode(wr, im)
}
