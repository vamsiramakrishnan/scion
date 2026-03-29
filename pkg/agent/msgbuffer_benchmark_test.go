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

package agent

import (
	"testing"
	"time"
)

// BenchmarkMessageBufferSend measures the cost of queuing a message
// (buffer insertion + timer management).
func BenchmarkMessageBufferSend(b *testing.B) {
	mb := NewMessageBuffer(2*time.Second, func(agentID, message string, interrupt bool) error {
		return nil // no-op delivery
	})
	defer mb.Close()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mb.Send("agent-1", "Hello, this is a test message for benchmarking")
	}
}

// BenchmarkMessageBufferSendMultipleAgents measures queuing messages for
// different agents (tests map access pattern).
func BenchmarkMessageBufferSendMultipleAgents(b *testing.B) {
	mb := NewMessageBuffer(2*time.Second, func(agentID, message string, interrupt bool) error {
		return nil
	})
	defer mb.Close()

	agents := []string{"agent-1", "agent-2", "agent-3", "agent-4", "agent-5"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mb.Send(agents[i%len(agents)], "Test message")
	}
}
