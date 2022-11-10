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
	"github.com/tidwall/gjson"
	"github.com/whosonfirst/go-ioutil"
	"image"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	// "strings"
	"time"
)

const SLACK_API_UPLOAD string = "https://slack.com/api/files.upload"

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
		channel:     channel,
		token:       token,
		encoder:     enc,
		logger:      logger,
	}

	return br, nil
}

func (b *SlackBroadcaster) BroadcastMessage(ctx context.Context, msg *broadcaster.Message) (uid.UID, error) {

	type upload_rsp struct {
		index int
		url string
	}

	var image_urls []string
	
	if len(msg.Images) > 0 {

		image_urls = make([]string, len(msg.Images))

		done_ch := make(chan bool)
		err_ch := make(chan error)
		upload_ch := make(chan upload_rsp)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		
		for idx, im := range msg.Images {
			
			go func(idx int, im image.Image){

				defer func(){
					done_ch <- true
				}()

				select {
				case <- ctx.Done():
					return
				default:
					// pass
				}
								
				url, err := b.uploadImage(ctx, im, args)

				if err != nil {
					err_ch <- fmt.Errorf("Failed to upload image, %w", err)
					return
				}

				upload_ch <- upload_rsp{ index: idx, url: url }
				return
				
			}(idx, im)

		}

		remaining := len(msg.Images)

		for remaining > 0 {
			select {
			case <- done_ch:
				remaining -= 1
			case err := <- err_ch:
				return nil, fmt.Errorf("Failed to broadcast message, %w", err)
			case rsp := <- upload_ch:
				image_urls[rsp.index] = rsp.url
			}
		}
		
	}

	// https://api.slack.com/methods/chat.postMessage/test
	// https://stackoverflow.com/questions/59939261/send-multiple-files-to-slack-via-api
	
	return uid.NewStringUID(ctx, "")
}

func (b *SlackBroadcaster) SetLogger(ctx context.Context, logger *log.Logger) error {
	b.logger = logger
	return nil
}

func (b *SlackBroadcaster) uploadImage(ctx context.Context, im image.Image) (string, error) {
	
	// comment := fmt.Sprintf("%s %s", msg.Title, msg.Body)
	// comment = strings.TrimSpace(comment)
	
	args := &url.Values{}
	// args.Set("channels", b.channel)
	// args.Set("initial_comment", comment)
	
	var buf bytes.Buffer
	wr := bufio.NewWriter(&buf)
	
	err := b.encoder.Encode(ctx, im, wr)
	
	if err != nil {
		return fmt.Errorf("Failed to encode image, %w", err)
	}
	
	wr.Flush()
	
	br := bytes.NewReader(buf.Bytes())
	return b.uploadReader(ctx, br, args)
}

func (b *SlackBroadcaster) uploadReader(ctx context.Context, r io.Reader, args *url.Values) (string, error) {

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
		return "", fmt.Errorf("Failed to create new request, %w", err)
	}

	req.Header.Add("Content-Type", wr.FormDataContentType())

	rsp, err := b.call(ctx, req)

	if err != nil {
		return "", fmt.Errorf("Failed to call the Slack API, %w", err)
	}

	defer rsp.Close()

	select {
	case err := <-err_ch:
		return "", fmt.Errorf("There was a problem upload your file, %w", err)
	default:
		//
	}

	body, err := io.ReadAll(rsp)

	if err != nil {
		return "", fmt.Errorf("Failed to read upload response body, %w", err)
	}

	url_rsp := gjson.GetBytes(body, "file.private_url")

	if !url_rsp.Exists() {
		return "", fmt.Errorf("Failed to determine upload (private) URL")
	}

	return url_rsp.String(), nil
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
