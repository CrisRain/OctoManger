package grpccodec

import (
	"math"
	"testing"

	"google.golang.org/grpc/encoding"
)

type samplePayload struct {
	Message string `json:"message"`
}

func TestCodecRegistration(t *testing.T) {
	if encoding.GetCodec(Name) == nil {
		t.Fatalf("expected codec to be registered")
	}
	if Codec().Name() != Name {
		t.Fatalf("unexpected codec name %q", Codec().Name())
	}
}

func TestJSONCodecMarshalUnmarshal(t *testing.T) {
	codec := jsonCodec{}
	data, err := codec.Marshal(samplePayload{Message: "hello"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded samplePayload
	if err := codec.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if decoded.Message != "hello" {
		t.Fatalf("unexpected decoded message %q", decoded.Message)
	}
}

func TestJSONCodecErrors(t *testing.T) {
	codec := jsonCodec{}
	if _, err := codec.Marshal(math.Inf(1)); err == nil {
		t.Fatalf("expected marshal error")
	}
	if err := codec.Unmarshal([]byte("not-json"), &samplePayload{}); err == nil {
		t.Fatalf("expected unmarshal error")
	}
}
