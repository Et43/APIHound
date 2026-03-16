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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Manifest is the top-level structure of an APIHound collector manifest file.
// It maps OpenAPI/Swagger specification files to operational/business units.
type Manifest struct {
	// OpCos is the list of operational/business units and their API specs.
	OpCos []OpCo `json:"opcos" yaml:"opcos"`
}

// OpCo represents an operational or business unit that owns one or more API
// specification files.
type OpCo struct {
	// Name is the unique name of the operational/business unit.
	Name string `json:"name" yaml:"name"`

	// Description is an optional human-readable description.
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Specs lists the API specification files belonging to this OpCo.
	Specs []SpecEntry `json:"specs" yaml:"specs"`
}

// SpecEntry represents a single OpenAPI/Swagger specification file within an OpCo.
type SpecEntry struct {
	// Path is the filesystem path to the specification file, relative to the manifest.
	Path string `json:"path" yaml:"path"`

	// ServiceName is an optional override for the service identifier.
	// When empty, the filename (without extension) is used.
	ServiceName string `json:"service_name,omitempty" yaml:"service_name,omitempty"`
}

// LoadFromFile reads and parses a manifest file. It supports both JSON and YAML
// based on the file extension.
func LoadFromFile(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading manifest file: %w", err)
	}

	var manifest Manifest

	extension := filepath.Ext(path)
	switch extension {
	case ".json":
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parsing JSON manifest: %w", err)
		}
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("parsing YAML manifest: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported manifest format %q (use .json, .yaml, or .yml)", extension)
	}

	return &manifest, nil
}

// Validate checks the manifest for structural correctness — all spec file paths
// are resolved relative to the manifest file's directory and verified to exist.
func (s *Manifest) Validate(manifestDir string) error {
	if len(s.OpCos) == 0 {
		return fmt.Errorf("manifest contains no OpCos")
	}

	for opCoIndex, opCo := range s.OpCos {
		if opCo.Name == "" {
			return fmt.Errorf("opco at index %d has no name", opCoIndex)
		}

		if len(opCo.Specs) == 0 {
			return fmt.Errorf("opco %q has no specs", opCo.Name)
		}

		for specIndex, spec := range opCo.Specs {
			if spec.Path == "" {
				return fmt.Errorf("opco %q spec at index %d has no path", opCo.Name, specIndex)
			}

			resolvedPath := spec.Path
			if !filepath.IsAbs(resolvedPath) {
				resolvedPath = filepath.Join(manifestDir, resolvedPath)
			}

			if _, err := os.Stat(resolvedPath); err != nil {
				return fmt.Errorf("opco %q spec %q: %w", opCo.Name, spec.Path, err)
			}
		}
	}

	return nil
}

// ResolveSpecPath resolves a spec path relative to the manifest directory,
// returning an absolute path.
func ResolveSpecPath(manifestDir string, specPath string) string {
	if filepath.IsAbs(specPath) {
		return specPath
	}
	return filepath.Join(manifestDir, specPath)
}
