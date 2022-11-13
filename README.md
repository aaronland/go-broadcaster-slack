# go-broadcaster-slack

Go package implementing the `aaronland/go-broadcaster` interfaces for broadcasting messages to Slack.

## Documentation

Documentation is incomplete at this time.

## Tools

```
$> make cli
go build -mod vendor -o bin/broadcast cmd/broadcast/main.go
```

### broadcast

```
$> bin/broadcast/main.go \
	-body 'this is a test' \
	-broadcaster 'slack://{SLACK_CHANNEL_NAME_OR_ID}?credentials={RUNTIMVAR_URI}' \
	-image /usr/local/000.jpg \
	-image /usr/local/001.jpg	
```

Where the `?credentials=` is expected to be a valid [sfomuseum/runtimevar](https://github.com/sfomuseum/runtimevar) URI. For example `file:///usr/local/slack-broadcaster.txt`.

The value of that URI, when dereferenced, is expected to contain a valid Slack API OAuth token. That token should have the following scopes: `channels:read`, `chat:write`, `files:write`.

#### Implementation details

Because of the way the Slack API works (I think) if a "broadcast" message contains no images it is posted using the `chat.postMessage` API method. If it contains images the message will be posted using the `files.upload` API method.

If a "broadcast" message contains images each image will be posted separately. Any text associated with the "broadcast" message will be assigned to the first image upload. If there's a way to upload multiple images with a single "chat" message using the API I haven't been to figure it out and I would welcome suggestions.

## See also

* https://github.com/aaronland/go-broadcaster
* https://github.com/sfomuseum/runtimevar

* https://api.slack.com/methods/files.upload
* https://api.slack.com/methods/chat.postMessage