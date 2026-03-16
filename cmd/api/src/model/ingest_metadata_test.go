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

package model_test

import (
	"testing"

	"github.com/specterops/bloodhound/cmd/api/src/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIngestTaskMetadata_Value(t *testing.T) {
	metadata := model.IngestTaskMetadata{
		OpCoName:        "Payments",
		OpCoDescription: "Payment processing unit",
	}

	value, err := metadata.Value()
	require.NoError(t, err)

	stringValue, ok := value.(string)
	require.True(t, ok)
	assert.Contains(t, stringValue, `"opco_name":"Payments"`)
	assert.Contains(t, stringValue, `"opco_description":"Payment processing unit"`)
}

func TestIngestTaskMetadata_Value_Empty(t *testing.T) {
	metadata := model.IngestTaskMetadata{}

	value, err := metadata.Value()
	require.NoError(t, err)

	stringValue, ok := value.(string)
	require.True(t, ok)
	assert.Equal(t, `{}`, stringValue)
}

func TestIngestTaskMetadata_Scan_Bytes(t *testing.T) {
	input := []byte(`{"opco_name":"Identity","opco_description":"IAM team"}`)

	var metadata model.IngestTaskMetadata
	err := metadata.Scan(input)
	require.NoError(t, err)

	assert.Equal(t, "Identity", metadata.OpCoName)
	assert.Equal(t, "IAM team", metadata.OpCoDescription)
}

func TestIngestTaskMetadata_Scan_String(t *testing.T) {
	input := `{"opco_name":"Finance"}`

	var metadata model.IngestTaskMetadata
	err := metadata.Scan(input)
	require.NoError(t, err)

	assert.Equal(t, "Finance", metadata.OpCoName)
	assert.Empty(t, metadata.OpCoDescription)
}

func TestIngestTaskMetadata_Scan_Nil(t *testing.T) {
	var metadata model.IngestTaskMetadata
	err := metadata.Scan(nil)
	require.NoError(t, err)

	assert.Empty(t, metadata.OpCoName)
}

func TestIngestTaskMetadata_Scan_InvalidType(t *testing.T) {
	var metadata model.IngestTaskMetadata
	err := metadata.Scan(42)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
}

func TestIngestTaskMetadata_RoundTrip(t *testing.T) {
	original := model.IngestTaskMetadata{
		OpCoName:        "Platform Engineering",
		OpCoDescription: "Core platform services",
	}

	value, err := original.Value()
	require.NoError(t, err)

	var restored model.IngestTaskMetadata
	err = restored.Scan(value)
	require.NoError(t, err)

	assert.Equal(t, original.OpCoName, restored.OpCoName)
	assert.Equal(t, original.OpCoDescription, restored.OpCoDescription)
}
