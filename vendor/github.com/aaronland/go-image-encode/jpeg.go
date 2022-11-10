package encode

import (
	"context"
	"image"
	"image/jpeg"
	"io"
)

type JPEGEncoder struct {
	Encoder
	options *jpeg.Options
}

func init() {

	ctx := context.Background()
	err := RegisterEncoder(ctx, NewJPEGEncoder, "jpg", "jpeg")

	if err != nil {
		panic(err)
	}
}

func NewJPEGEncoder(ctx context.Context, uri string) (Encoder, error) {

	opts := &jpeg.Options{Quality: 100}

	e := &JPEGEncoder{
		options: opts,
	}

	return e, nil
}

func (e *JPEGEncoder) MimeType() string {
	return "image/jpeg"
}

func (e *JPEGEncoder) Extension() string {
	return ".jpg"
}

func (e *JPEGEncoder) Encode(ctx context.Context, im image.Image, wr io.Writer) error {
	return jpeg.Encode(wr, im, e.options)
}
