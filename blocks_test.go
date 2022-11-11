package slack

import (
	"encoding/json"
	"testing"
)

func TestBlocks(t *testing.T) {

	expected := `[{"type":"section","text":{"type":"mrkdwn","text":"Testing"}},{"type":"section","accessory":{"type":"image","image_url":"https://files.slack.com/files-pri/testing.jpg"}}]`

	msg_text := "Testing"
	image_urls := []string{"https://files.slack.com/files-pri/testing.jpg"}

	blocks := make([]Block, 0)

	if msg_text != "" {

		text_block := Block{
			Type: "section",
			Text: &Text{
				Type: "mrkdwn",
				Text: msg_text,
			},
		}

		blocks = append(blocks, text_block)
	}

	for _, url := range image_urls {

		img_block := Block{
			Type: "section",
			Accessory: &Accessory{
				Type:     "image",
				ImageURL: url,
			},
		}

		blocks = append(blocks, img_block)
	}

	enc_blocks, err := json.Marshal(blocks)

	if err != nil {
		t.Fatalf("Failed to marshal blocks, %v", err)
	}

	str_blocks := string(enc_blocks)

	if str_blocks != expected {
		t.Fatalf("Unexpected output '%s'", str_blocks)
	}
}
