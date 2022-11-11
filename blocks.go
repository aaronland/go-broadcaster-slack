package slack

type Block struct {
	Type      string `json:"type"`
	Text      *Text  `json:"text,omitempty"`
	Accessory *Image `json:"accessory,omitempty"`
}

type Text struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type Image struct {
	Type     string `json:"type,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	AltText  string `json:"alt_text,omitempty"`
}
