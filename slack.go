package slack

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"github.com/aaronland/go-broadcaster"
	"github.com/aaronland/go-image-encode"
	"github.com/aaronland/go-uid"
	"github.com/sfomuseum/runtimevar"
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-ioutil"
	"image"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const SLACK_API_UPLOAD string = "https://slack.com/api/files.upload"
const SLACK_API_CHAT string = "https://slack.com/api/chat.postMessage"

func init() {
	ctx := context.Background()
	broadcaster.RegisterBroadcaster(ctx, "slack", NewSlackBroadcaster)
}

type SlackBroadcaster struct {
	broadcaster.Broadcaster
	http_client *http.Client
	channel     string
	token       string
	encoder     encode.Encoder
	logger      *log.Logger
}

func NewSlackBroadcaster(ctx context.Context, uri string) (broadcaster.Broadcaster, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	channel := u.Host

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

	enc, err := encode.NewEncoder(ctx, "png://")

	if err != nil {
		return nil, fmt.Errorf("Failed to create image encoder, %w", err)
	}

	http_client := &http.Client{}
	logger := log.Default()

	br := &SlackBroadcaster{
		http_client: http_client,
		channel:     channel,
		token:       token,
		encoder:     enc,
		logger:      logger,
	}

	return br, nil
}

func (br *SlackBroadcaster) BroadcastMessage(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	if len(msg.Images) > 0 {
		return br.broadcastMessageWithImages(ctx, msg)
	}

	return br.broadcastMessage(ctx, msg)
}

func (br *SlackBroadcaster) broadcastMessage(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	msg_text := fmt.Sprintf("%s %s", msg.Title, msg.Body)
	msg_text = strings.TrimSpace(msg_text)

	args := url.Values{}
	args.Set("channel", br.channel)
	args.Set("text", msg_text)

	args_enc := args.Encode()
	args_r := strings.NewReader(args_enc)

	req, err := http.NewRequest("POST", SLACK_API_CHAT, args_r)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new chat request, %w", err)
	}

	req.Header.Set("Content-type", "application/x-www-form-urlencoded")

	rsp, err := br.call(ctx, req)

	if err != nil {
		return nil, fmt.Errorf("Failed to call the Slack API, %w", err)
	}

	body, err := io.ReadAll(rsp)

	if err != nil {
		return nil, fmt.Errorf("Failed to read API response, %w", err)
	}

	ok_rsp := gjson.GetBytes(body, "ok")

	if !ok_rsp.Bool() {
		err_rsp := gjson.GetBytes(body, "error")
		// something something something list errors
		return nil, fmt.Errorf("API returned an error, %s", err_rsp.String())
	}

	// there isn't really an ID property in these responses
	return br.uid(ctx)
}

func (br *SlackBroadcaster) broadcastMessageWithImages(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	for idx, im := range msg.Images {

		args := &url.Values{}
		args.Set("channels", br.channel)

		if idx == 0 {
			msg_text := fmt.Sprintf("%s %s", msg.Title, msg.Body)
			msg_text = strings.TrimSpace(msg_text)
			args.Set("text", msg_text)
		}

		err := br.uploadImage(ctx, im, args)

		if err != nil {
			return nil, fmt.Errorf("Failed to upload image, %w", err)
		}

	}

	// Unlike the chat API there are ID properties in file upload responses but
	// ultimately they are not relevant for doing anything with the message that
	// is posted to a channel
	return br.uid(ctx)
}

func (br *SlackBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {
	br.logger = logger
	return nil
}

func (br *SlackBroadcaster) uploadImage(ctx context.Context, im image.Image, args *url.Values) error {

	var buf bytes.Buffer
	wr := bufio.NewWriter(&buf)

	err := br.encoder.Encode(ctx, im, wr)

	if err != nil {
		return fmt.Errorf("Failed to encode image, %w", err)
	}

	wr.Flush()

	b_r := bytes.NewReader(buf.Bytes())
	return br.uploadReader(ctx, b_r, args)
}

func (br *SlackBroadcaster) uploadReader(ctx context.Context, r io.Reader, args *url.Values) error {

	pipe_r, pipe_wr := io.Pipe()

	wr := multipart.NewWriter(pipe_wr)

	err_ch := make(chan error)

	go func() {

		defer pipe_wr.Close()

		ioWriter, err := wr.CreateFormFile("file", "upload.png")

		if err != nil {
			err_ch <- fmt.Errorf("Failed to create form, %w", err)
			return
		}

		_, err = io.Copy(ioWriter, r)

		if err != nil {
			err_ch <- fmt.Errorf("Failed to copy file, %w", err)
			return
		}

		for key, val := range *args {
			_ = wr.WriteField(key, val[0])
		}

		err = wr.Close()

		if err != nil {
			err_ch <- fmt.Errorf("Failed to close upload writer, %w", err)
			return
		}
	}()

	req, err := http.NewRequest("POST", SLACK_API_UPLOAD, pipe_r)

	if err != nil {
		return fmt.Errorf("Failed to create new request, %w", err)
	}

	req.Header.Add("Content-Type", wr.FormDataContentType())

	rsp, err := br.call(ctx, req)

	if err != nil {
		return fmt.Errorf("Failed to call the Slack API, %w", err)
	}

	defer rsp.Close()

	select {
	case err := <-err_ch:
		return fmt.Errorf("There was a problem upload your file, %w", err)
	default:
		//
	}

	return nil
}

func (br *SlackBroadcaster) call(ctx context.Context, req *http.Request) (io.ReadSeekCloser, error) {

	req = req.WithContext(ctx)

	bearer_token := fmt.Sprintf("Bearer %s", br.token)

	req.Header.Set("Authorization", bearer_token)
	rsp, err := br.http_client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("Failed to execute HTTP request, %w", err)
	}

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API call failed with status '%s'", rsp.Status)
	}

	return ioutil.NewReadSeekCloser(rsp.Body)
}

func (br *SlackBroadcaster) uid(ctx context.Context) (uid.UID, error) {
	now := time.Now()
	ts := now.Unix()
	return uid.NewInt64UID(ctx, ts)
}
