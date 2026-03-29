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

import "regexp"

var (
	// validGCSBucket matches valid GCS bucket names per Google Cloud naming rules.
	validGCSBucket = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,220}[a-z0-9]$`)
	// validGCSPath matches safe GCS object path prefixes (no shell metacharacters).
	validGCSPath = regexp.MustCompile(`^[a-zA-Z0-9/_.\-]+$`)
)

// isValidGCSBucket returns true if the bucket name is safe for use in shell commands.
func isValidGCSBucket(name string) bool {
	return validGCSBucket.MatchString(name)
}

// isValidGCSPath returns true if the GCS path/prefix is safe for use in shell commands.
func isValidGCSPath(path string) bool {
	return validGCSPath.MatchString(path)
}
