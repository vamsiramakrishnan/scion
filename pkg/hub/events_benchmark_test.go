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

package hub

import (
	"context"
	"testing"

	"github.com/GoogleCloudPlatform/scion/pkg/store"
)

// BenchmarkPublishAgentStatus measures the cost of publishing an agent status
// event including JSON marshaling and channel fan-out.
func BenchmarkPublishAgentStatus(b *testing.B) {
	pub := NewChannelEventPublisher()
	defer pub.Close()

	// Add a subscriber to exercise the fan-out path
	ch, unsub := pub.Subscribe("grove.>")
	defer unsub()

	// Drain subscriber channel in background
	go func() {
		for range ch {
		}
	}()

	agent := &store.Agent{
		ID:              "agent-1",
		GroveID:         "grove-1",
		Phase:           "running",
		Activity:        "coding",
		ContainerStatus: "running",
		ToolName:        "read_file",
		Message:         "Reading main.go",
		TaskSummary:     "Implementing feature X",
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pub.PublishAgentStatus(ctx, agent)
	}
}

// BenchmarkPublishNoSubscribers measures publish cost when no subscribers match.
func BenchmarkPublishNoSubscribers(b *testing.B) {
	pub := NewChannelEventPublisher()
	defer pub.Close()

	agent := &store.Agent{
		ID:      "agent-1",
		GroveID: "grove-1",
		Phase:   "running",
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pub.PublishAgentStatus(ctx, agent)
	}
}

// BenchmarkPublishMultipleSubscribers measures fan-out cost with 10 subscribers.
func BenchmarkPublishMultipleSubscribers(b *testing.B) {
	pub := NewChannelEventPublisher()
	defer pub.Close()

	// Add 10 subscribers
	for i := 0; i < 10; i++ {
		ch, unsub := pub.Subscribe("grove.>")
		defer unsub()
		go func() {
			for range ch {
			}
		}()
	}

	agent := &store.Agent{
		ID:      "agent-1",
		GroveID: "grove-1",
		Phase:   "running",
	}

	ctx := context.Background()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pub.PublishAgentStatus(ctx, agent)
	}
}

// BenchmarkSubjectMatchesPattern measures the cost of NATS-style pattern matching.
func BenchmarkSubjectMatchesPattern(b *testing.B) {
	b.Run("exact", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			subjectMatchesPattern("grove.abc.agent.status", "grove.abc.agent.status")
		}
	})
	b.Run("wildcard_star", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			subjectMatchesPattern("grove.*.agent.status", "grove.abc.agent.status")
		}
	})
	b.Run("wildcard_gt", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			subjectMatchesPattern("grove.>", "grove.abc.agent.status")
		}
	})
	b.Run("no_match", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			subjectMatchesPattern("broker.>", "grove.abc.agent.status")
		}
	})
}
