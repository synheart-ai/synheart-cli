package encoding

import (
	"encoding/json"

	"github.com/synheart/synheart-cli/internal/models"
)

// Format represents the encoding format
type Format string

const (
	FormatJSON     Format = "json"
	FormatProtobuf Format = "protobuf"
)

// Encoder encodes events to bytes
type Encoder interface {
	Encode(event models.Event) ([]byte, error)
	ContentType() string
}

// JSONEncoder encodes events as JSON
type JSONEncoder struct{}

func NewJSONEncoder() *JSONEncoder {
	return &JSONEncoder{}
}

func (e *JSONEncoder) Encode(event models.Event) ([]byte, error) {
	return json.Marshal(event)
}

func (e *JSONEncoder) ContentType() string {
	return "application/json"
}

// NewEncoder creates an encoder for the given format
func NewEncoder(format Format) Encoder {
	switch format {
	case FormatProtobuf:
		return NewProtobufEncoder()
	default:
		return NewJSONEncoder()
	}
}
