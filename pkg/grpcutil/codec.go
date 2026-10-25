package grpcutil

import (
	"encoding/json"

	"google.golang.org/grpc/encoding"
)

// JSONCodec implements google.golang.org/grpc/encoding.Codec using encoding/json.
// It is registered under the explicit content subtype "json" to support custom
// struct-based RPCs without replacing or corrupting the standard "proto" codec.
type JSONCodec struct{}

func (JSONCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (JSONCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (JSONCodec) Name() string {
	return "json"
}

func init() {
	encoding.RegisterCodec(JSONCodec{})
}
