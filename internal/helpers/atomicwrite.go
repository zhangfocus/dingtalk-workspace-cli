// Copyright 2026 Alibaba Group
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

package helpers

import (
	"io"
	"os"

	"github.com/DingTalk-Real-AI/dingtalk-workspace-cli/internal/fileutil"
)

// AtomicWrite writes data to path atomically by creating a temp file in the
// same directory, writing and fsyncing the data, then renaming over the target.
// It replaces os.WriteFile for all config and download file writes.
//
// os.WriteFile truncates the target before writing, so a process kill (CI timeout,
// OOM, Ctrl+C) between truncate and completion leaves the file empty or partial.
// AtomicWrite avoids this: on any failure the temp file is cleaned up and the
// original file remains untouched.
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	return fileutil.AtomicWrite(path, data, perm)
}

// AtomicWriteFromReader atomically copies reader contents into path.
func AtomicWriteFromReader(path string, reader io.Reader, perm os.FileMode) (int64, error) {
	return fileutil.AtomicWriteFromReader(path, reader, perm)
}

// AtomicWriteJSON is a convenience wrapper for writing JSON data atomically.
// It uses 0600 permissions by default for sensitive data.
func AtomicWriteJSON(path string, data []byte) error {
	return fileutil.AtomicWriteJSON(path, data)
}
