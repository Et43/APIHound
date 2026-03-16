// Copyright 2026 Specter Ops, Inc.
//
// Licensed under the Apache License, Version 2.0
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
//
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadFromFile_YAML(t *testing.T) {
	content := `opcos:
  - name: "Payments"
    description: "Payment processing"
    specs:
      - path: "./specs/payments.yaml"
  - name: "Identity"
    specs:
      - path: "./specs/identity.json"
        service_name: "identity-svc"
`
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.yaml")
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0644))

	collectorManifest, err := LoadFromFile(manifestPath)
	require.NoError(t, err)

	assert.Len(t, collectorManifest.OpCos, 2)
	assert.Equal(t, "Payments", collectorManifest.OpCos[0].Name)
	assert.Equal(t, "Payment processing", collectorManifest.OpCos[0].Description)
	assert.Len(t, collectorManifest.OpCos[0].Specs, 1)
	assert.Equal(t, "./specs/payments.yaml", collectorManifest.OpCos[0].Specs[0].Path)

	assert.Equal(t, "Identity", collectorManifest.OpCos[1].Name)
	assert.Equal(t, "identity-svc", collectorManifest.OpCos[1].Specs[0].ServiceName)
}

func TestLoadFromFile_JSON(t *testing.T) {
	content := `{
  "opcos": [
    {
      "name": "Payments",
      "specs": [{ "path": "./api.json" }]
    }
  ]
}`
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.json")
	require.NoError(t, os.WriteFile(manifestPath, []byte(content), 0644))

	collectorManifest, err := LoadFromFile(manifestPath)
	require.NoError(t, err)

	assert.Len(t, collectorManifest.OpCos, 1)
	assert.Equal(t, "Payments", collectorManifest.OpCos[0].Name)
}

func TestLoadFromFile_UnsupportedFormat(t *testing.T) {
	tempDir := t.TempDir()
	manifestPath := filepath.Join(tempDir, "manifest.txt")
	require.NoError(t, os.WriteFile(manifestPath, []byte("hello"), 0644))

	_, err := LoadFromFile(manifestPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported manifest format")
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/manifest.yaml")
	assert.Error(t, err)
}

func TestManifest_Validate_Valid(t *testing.T) {
	tempDir := t.TempDir()

	// Create a spec file that the manifest references
	specPath := filepath.Join(tempDir, "api.yaml")
	require.NoError(t, os.WriteFile(specPath, []byte("openapi: 3.0.0"), 0644))

	collectorManifest := &Manifest{
		OpCos: []OpCo{
			{
				Name: "TestOpCo",
				Specs: []SpecEntry{
					{Path: "api.yaml"},
				},
			},
		},
	}

	assert.NoError(t, collectorManifest.Validate(tempDir))
}

func TestManifest_Validate_EmptyOpCos(t *testing.T) {
	collectorManifest := &Manifest{}
	assert.Error(t, collectorManifest.Validate("/tmp"))
}

func TestManifest_Validate_EmptyOpCoName(t *testing.T) {
	collectorManifest := &Manifest{
		OpCos: []OpCo{
			{Name: "", Specs: []SpecEntry{{Path: "x.json"}}},
		},
	}
	err := collectorManifest.Validate("/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no name")
}

func TestManifest_Validate_EmptySpecs(t *testing.T) {
	collectorManifest := &Manifest{
		OpCos: []OpCo{
			{Name: "Test", Specs: []SpecEntry{}},
		},
	}
	err := collectorManifest.Validate("/tmp")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "has no specs")
}

func TestManifest_Validate_MissingSpecFile(t *testing.T) {
	tempDir := t.TempDir()
	collectorManifest := &Manifest{
		OpCos: []OpCo{
			{
				Name:  "Test",
				Specs: []SpecEntry{{Path: "nonexistent.yaml"}},
			},
		},
	}

	err := collectorManifest.Validate(tempDir)
	assert.Error(t, err)
}

func TestResolveSpecPath_Relative(t *testing.T) {
	result := ResolveSpecPath("/home/user/project", "specs/api.yaml")
	assert.Equal(t, "/home/user/project/specs/api.yaml", result)
}

func TestResolveSpecPath_Absolute(t *testing.T) {
	result := ResolveSpecPath("/home/user/project", "/absolute/specs/api.yaml")
	assert.Equal(t, "/absolute/specs/api.yaml", result)
}
