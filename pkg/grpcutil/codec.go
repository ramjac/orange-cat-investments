package grpcutil

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// JSONCodec implements google.golang.org/grpc/encoding.Codec using encoding/json.
// It is registered under name "proto" to serve as the default wire codec for gRPC calls
// while supporting generic Go structs, as well as under "json".
type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JSONCodec) Name() string {
	return "proto"
}

// JSONCodecExplicit has Name() "json"
type JSONCodecExplicit struct{}

func (JSONCodecExplicit) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodecExplicit) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JSONCodecExplicit) Name() string {
	return "json"
}

func init() {
	encoding.RegisterCodec(JSONCodec{})
	encoding.RegisterCodec(JSONCodecExplicit{})
}
