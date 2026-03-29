// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wsprotocol

import (
	"context"
	"encoding/json"
	"testing"
)

// BenchmarkParseEnvelope measures the cost of parsing a WebSocket message envelope.
func BenchmarkParseEnvelope(b *testing.B) {
	data := []byte(`{"type":"stream","streamId":"s-1234","data":"SGVsbG8gV29ybGQ="}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseEnvelope(data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseStreamFrame measures full deserialization of a stream frame.
func BenchmarkParseStreamFrame(b *testing.B) {
	data := []byte(`{"type":"stream","streamId":"s-1234","data":"SGVsbG8gV29ybGQ="}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage[StreamFrame](data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalStreamFrame measures serialization of a stream frame.
func BenchmarkMarshalStreamFrame(b *testing.B) {
	frame := NewStreamFrame("s-1234", make([]byte, 4096))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(frame)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalPTYData measures serialization of a PTY data message
// with a typical 32KB payload (the ptyMaxDataSize).
func BenchmarkMarshalPTYData(b *testing.B) {
	msg := NewPTYDataMessage(make([]byte, 32*1024))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMarshalRequestEnvelope measures serialization of a tunneled HTTP request.
func BenchmarkMarshalRequestEnvelope(b *testing.B) {
	env := NewRequestEnvelope("req-1", "POST", "/api/v1/agents", "",
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token-xxxx",
		},
		[]byte(`{"name":"test-agent","grove":"my-grove"}`),
	)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(env)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseRequestEnvelope measures deserialization of a tunneled HTTP request.
func BenchmarkParseRequestEnvelope(b *testing.B) {
	env := NewRequestEnvelope("req-1", "POST", "/api/v1/agents", "",
		map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token-xxxx",
		},
		[]byte(`{"name":"test-agent","grove":"my-grove"}`),
	)
	data, _ := json.Marshal(env)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ParseMessage[RequestEnvelope](data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkBufferPool measures the overhead of sync.Pool for buffer reuse.
func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := GetBuffer()
		*buf = append(*buf, "test data for pool benchmark"...)
		PutBuffer(buf)
	}
}

// BenchmarkMessageRouterRoute measures the cost of routing a message through
// the MessageRouter (map lookup + envelope parsing).
func BenchmarkMessageRouterRoute(b *testing.B) {
	router := NewMessageRouter()
	router.Handle(TypeStream, func(_ context.Context, _ *Connection, _ []byte) error {
		return nil
	})
	router.Handle(TypePing, func(_ context.Context, _ *Connection, _ []byte) error {
		return nil
	})
	data := []byte(`{"type":"stream","streamId":"s-1"}`)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = router.Route(nil, nil, data)
	}
}
