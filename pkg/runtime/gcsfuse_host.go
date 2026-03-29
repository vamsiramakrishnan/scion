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

package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// GCSFuseMount represents a host-side GCS FUSE mount.
type GCSFuseMount struct {
	Bucket     string
	Prefix     string
	MountPoint string // host-side mount point
}

// GCSFuseCleanup tracks host-side FUSE mounts for cleanup.
type GCSFuseCleanup struct {
	mu     sync.Mutex
	mounts []GCSFuseMount
}

// MountGCSOnHost mounts a GCS bucket on the host using gcsfuse.
// Returns the host-side mount point path. Caller must call Cleanup() when done.
func (c *GCSFuseCleanup) MountGCSOnHost(ctx context.Context, bucket, prefix string) (string, error) {
	// Create host-side mount point
	mountDir, err := os.MkdirTemp("", fmt.Sprintf("scion-gcs-%s-*", sanitizeBucketName(bucket)))
	if err != nil {
		return "", fmt.Errorf("create GCS mount dir: %w", err)
	}

	// Build gcsfuse command
	args := []string{"--implicit-dirs"}
	if prefix != "" {
		args = append(args, "--only-dir", prefix)
	}
	args = append(args, bucket, mountDir)

	cmd := exec.CommandContext(ctx, "gcsfuse", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(mountDir)
		return "", fmt.Errorf("gcsfuse mount %s: %w\n%s", bucket, err, string(out))
	}

	c.mu.Lock()
	c.mounts = append(c.mounts, GCSFuseMount{
		Bucket:     bucket,
		Prefix:     prefix,
		MountPoint: mountDir,
	})
	c.mu.Unlock()

	slog.Info("Mounted GCS bucket on host", "bucket", bucket, "prefix", prefix, "mountpoint", mountDir)
	return mountDir, nil
}

// Cleanup unmounts all GCS FUSE mounts and removes mount directories.
func (c *GCSFuseCleanup) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, m := range c.mounts {
		// Unmount FUSE
		if err := exec.Command("fusermount", "-u", m.MountPoint).Run(); err != nil {
			slog.Warn("Failed to unmount GCS FUSE", "mountpoint", m.MountPoint, "error", err)
			// Try lazy unmount as fallback
			exec.Command("fusermount", "-uz", m.MountPoint).Run()
		}
		os.RemoveAll(m.MountPoint)
	}
	c.mounts = nil
}

func sanitizeBucketName(bucket string) string {
	s := strings.ReplaceAll(bucket, "/", "-")
	if len(s) > 20 {
		s = s[:20]
	}
	return s
}
