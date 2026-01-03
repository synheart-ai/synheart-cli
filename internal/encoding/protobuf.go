package encoding

import (
	"github.com/synheart/synheart-cli/internal/models"
	"github.com/synheart/synheart-cli/internal/proto/hsi"
	"google.golang.org/protobuf/proto"
)

// ProtobufEncoder encodes events as protocol buffers
type ProtobufEncoder struct{}

func NewProtobufEncoder() *ProtobufEncoder {
	return &ProtobufEncoder{}
}

func (e *ProtobufEncoder) Encode(event models.Event) ([]byte, error) {
	pb := eventToProto(event)
	return proto.Marshal(pb)
}

func (e *ProtobufEncoder) ContentType() string {
	return "application/x-protobuf"
}

func eventToProto(e models.Event) *hsi.Event {
	pb := &hsi.Event{
		SchemaVersion: e.SchemaVersion,
		EventId:       e.EventID,
		Ts:            e.Timestamp,
		Source: &hsi.Source{
			Type: e.Source.Type,
			Id:   e.Source.ID,
		},
		Session: &hsi.Session{
			RunId:    e.Session.RunID,
			Scenario: e.Session.Scenario,
			Seed:     e.Session.Seed,
		},
		Signal: &hsi.Signal{
			Name:    e.Signal.Name,
			Unit:    e.Signal.Unit,
			Quality: e.Signal.Quality,
		},
		Meta: &hsi.Meta{
			Sequence: e.Meta.Sequence,
		},
	}

	if e.Source.Side != nil {
		pb.Source.Side = e.Source.Side
	}

	pb.Signal.Value = toSignalValue(e.Signal.Value)
	return pb
}

func toSignalValue(v interface{}) *hsi.SignalValue {
	switch val := v.(type) {
	case float64:
		return &hsi.SignalValue{Kind: &hsi.SignalValue_Scalar{Scalar: val}}
	case int:
		return &hsi.SignalValue{Kind: &hsi.SignalValue_Scalar{Scalar: float64(val)}}
	case string:
		return &hsi.SignalValue{Kind: &hsi.SignalValue_Text{Text: val}}
	case []float64:
		if len(val) >= 3 {
			return &hsi.SignalValue{Kind: &hsi.SignalValue_Vector{
				Vector: &hsi.Vector3{X: val[0], Y: val[1], Z: val[2]},
			}}
		}
	case []interface{}:
		// JSON unmarshals arrays as []interface{}
		if len(val) >= 3 {
			x, ok1 := toFloat(val[0])
			y, ok2 := toFloat(val[1])
			z, ok3 := toFloat(val[2])
			if ok1 && ok2 && ok3 {
				return &hsi.SignalValue{Kind: &hsi.SignalValue_Vector{
					Vector: &hsi.Vector3{X: x, Y: y, Z: z},
				}}
			}
		}
	}
	return nil
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	}
	return 0, false
}
