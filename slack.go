package slack

/*
 get channel list
 https://api.slack.com/methods/conversations.list/test

 > curl -F "file=@000.jpg" -F "initial_comment=Testing" -F channels={CHANNEL} -H "Authorization: Bearer {TOKEN}" https://slack.com/api/files.upload
{"ok":true,"file":{"id":"F04ADCSLQ2Y"
*/

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/aaronland/go-broadcaster"
	"github.com/aaronland/go-image-encode"
	"github.com/aaronland/go-uid"
	"github.com/sfomuseum/runtimevar"
	"mime/multipart"	
	_ "image"
	"log"
	"net/http"
	"net/url"
	"time"
	"io"
	"github.com/whosonfirst/go-ioutil"
	"os"
)

func init() {
	ctx := context.Background()
	broadcaster.RegisterBroadcaster(ctx, "slack", NewSlackBroadcaster)
}

type SlackBroadcaster struct {
	broadcaster.Broadcaster
	http_client        *http.Client
	channel string
	token string
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

	token, err := runtimevar.StringVar(rt_ctx, creds_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive URI from credentials, %w", err)
	}

	channel := u.Host
	
	enc, err := encode.NewEncoder(ctx, "png://")

	if err != nil {
		return nil, fmt.Errorf("Failed to create image encoder, %w", err)
	}

	http_client := &http.Client{}
	logger := log.Default()

	br := &SlackBroadcaster{
		http_client: http_client,
		channel: channel,
		token: token,
		encoder:       enc,
		logger:        logger,
	}

	return br, nil
}

func (b *SlackBroadcaster) BroadcastMessage(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	for _, im := range msg.Images {
		
		args := &url.Values{}
		args.Set("channels", b.channel)
		args.Set("initial_comment", msg.Body)

		var buf bytes.Buffer
		wr := bufio.NewWriter(&buf)
		
		err := b.encoder.Encode(ctx, im, wr)
		
		if err != nil {
			return nil, fmt.Errorf("Failed to encode image, %w", err)
		}

		wr.Flush()

		br := bytes.NewReader(buf.Bytes())
		
		rsp, err := b.upload(ctx, br, args)

		if err != nil {
			return nil, fmt.Errorf("Failed to upload image, %w", err)
		}

		io.Copy(os.Stdout, rsp)
	}

	return uid.NewStringUID(ctx, "")
}

func (b *SlackBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {
	b.logger = logger
	return nil
}

func (b *SlackBroadcaster) upload(ctx context.Context, r io.Reader, args *url.Values) (io.ReadSeekCloser, error) {

	pipeReader, pipeWriter := io.Pipe()
	
	wr := multipart.NewWriter(pipeWriter)
	
	errc := make(chan error)
	
	go func() {
		
		defer pipeWriter.Close()
		
		ioWriter, err := wr.CreateFormFile("file", "upload.png")
		
		if err != nil {
			errc <- err
			return
		}
		_, err = io.Copy(ioWriter, r)
		if err != nil {
			errc <- err
			return
		}
		if err = wr.Close(); err != nil {
			errc <- err
			return
		}
	}()
		
	req, err := http.NewRequest("POST", "https://slack.com/api/files.upload", pipeReader)

	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", wr.FormDataContentType())

	rsp, err := b.call(ctx, req)

	if err != nil {
		return nil, err
	}

	select {
	case err = <-errc:
		return nil, err
	default:
		return rsp, nil
	}
}

func (br *SlackBroadcaster) call(ctx context.Context, req *http.Request) (io.ReadSeekCloser, error) {

	req = req.WithContext(ctx)

	bearer_token := fmt.Sprintf("Bearer %s", br.token)
	
	req.Header.Set("Authorization", bearer_token)
	rsp, err := br.http_client.Do(req)

	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		rsp.Body.Close()
		return nil, fmt.Errorf("API call failed with status '%s'", rsp.Status)
	}

	return ioutil.NewReadSeekCloser(rsp.Body)
}
	
