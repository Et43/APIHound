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

package apiparser

import (
	"testing"

	"github.com/specterops/bloodhound/packages/go/ein"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvert_WithoutOpCo(t *testing.T) {
	doc := APIDoc{
		Title:       "Test API",
		Description: "A test API",
		Version:     "1.0.0",
		SpecVersion: "3.0.0",
		Servers:     []APIServer{{URL: "https://api.example.com"}},
		Endpoints: []APIEndpoint{
			{
				Path:   "/users",
				Method: "GET",
			},
		},
	}

	result := Convert(doc, "test-api", nil)

	// Should have nodes: service + server + endpoint = 3
	assert.GreaterOrEqual(t, len(result.Nodes), 3)

	// No OpCo node should exist
	for _, node := range result.Nodes {
		for _, kind := range node.Kinds {
			assert.NotEqual(t, KindAPIOpCo, kind, "should not contain APIOpCo node when opts is nil")
		}
	}

	// No BelongsToOpCo edge should exist
	for _, edge := range result.Edges {
		assert.NotEqual(t, EdgeBelongsToOpCo, edge.Kind, "should not contain BelongsToOpCo edge when opts is nil")
	}

	// Service node should not have opco_name property
	serviceNode := findNodeByKind(result, KindAPIService)
	require.NotNil(t, serviceNode)
	_, hasOpCo := serviceNode.Properties["opco_name"]
	assert.False(t, hasOpCo, "service node should not have opco_name when opts is nil")
}

func TestConvert_WithOpCo(t *testing.T) {
	doc := APIDoc{
		Title:       "Payments API",
		Description: "Payment processing",
		Version:     "2.0.0",
		SpecVersion: "3.0.1",
		Servers:     []APIServer{{URL: "https://payments.example.com"}},
		Endpoints: []APIEndpoint{
			{
				Path:   "/charge",
				Method: "POST",
			},
		},
	}

	opts := &ConvertOptions{
		OpCoName:        "Payments Division",
		OpCoDescription: "Handles all payment processing",
	}

	result := Convert(doc, "payments-api", opts)

	// Should have nodes: service + server + endpoint + opco = 4
	assert.GreaterOrEqual(t, len(result.Nodes), 4)

	// OpCo node should exist
	opCoNode := findNodeByKind(result, KindAPIOpCo)
	require.NotNil(t, opCoNode, "should contain an APIOpCo node")
	assert.Equal(t, "Payments Division", opCoNode.Properties["name"])
	assert.Equal(t, "Handles all payment processing", opCoNode.Properties["description"])

	// BelongsToOpCo edge should exist
	belongsEdge := findEdgeByKind(result, EdgeBelongsToOpCo)
	require.NotNil(t, belongsEdge, "should contain a BelongsToOpCo edge")
	assert.Equal(t, opCoNode.ID, belongsEdge.End.Value)

	// Service node should have opco_name property
	serviceNode := findNodeByKind(result, KindAPIService)
	require.NotNil(t, serviceNode)
	assert.Equal(t, "Payments Division", serviceNode.Properties["opco_name"])
}

func TestConvert_OpCoIdempotency(t *testing.T) {
	// Two different specs under the same OpCo should produce the same OpCo objectID
	opCoName := "Shared OpCo"

	doc1 := APIDoc{Title: "API A", SpecVersion: "3.0.0"}
	doc2 := APIDoc{Title: "API B", SpecVersion: "3.0.0"}

	opts := &ConvertOptions{OpCoName: opCoName}

	result1 := Convert(doc1, "api-a", opts)
	result2 := Convert(doc2, "api-b", opts)

	opCo1 := findNodeByKind(result1, KindAPIOpCo)
	opCo2 := findNodeByKind(result2, KindAPIOpCo)

	require.NotNil(t, opCo1)
	require.NotNil(t, opCo2)
	assert.Equal(t, opCo1.ID, opCo2.ID, "same OpCo name should produce the same objectID")
}

func TestConvert_OpCoWithEmptyName(t *testing.T) {
	doc := APIDoc{Title: "Test", SpecVersion: "3.0.0"}
	opts := &ConvertOptions{OpCoName: ""}

	result := Convert(doc, "test", opts)

	// Empty OpCo name should not create an OpCo node
	opCoNode := findNodeByKind(result, KindAPIOpCo)
	assert.Nil(t, opCoNode, "empty OpCo name should not create an APIOpCo node")
}

func TestConvert_OpCoWithoutDescription(t *testing.T) {
	doc := APIDoc{Title: "Test", SpecVersion: "3.0.0"}
	opts := &ConvertOptions{OpCoName: "NoDesc OpCo"}

	result := Convert(doc, "test", opts)

	opCoNode := findNodeByKind(result, KindAPIOpCo)
	require.NotNil(t, opCoNode)
	assert.Equal(t, "NoDesc OpCo", opCoNode.Properties["name"])
	_, hasDesc := opCoNode.Properties["description"]
	assert.False(t, hasDesc, "OpCo node should not have description when it's empty")
}

// findNodeByKind returns the first node matching the given kind, or nil.
func findNodeByKind(result ConvertResult, kind string) *nodeResult {
	for _, node := range result.Nodes {
		for _, nodeKind := range node.Kinds {
			if nodeKind == kind {
				return &nodeResult{
					ID:         node.ID,
					Kinds:      node.Kinds,
					Properties: node.Properties,
				}
			}
		}
	}
	return nil
}

// findEdgeByKind returns the first edge matching the given kind, or nil.
func findEdgeByKind(result ConvertResult, kind string) *edgeResult {
	for _, edge := range result.Edges {
		if edge.Kind == kind {
			return &edgeResult{
				Start:      edge.Start,
				End:        edge.End,
				Kind:       edge.Kind,
				Properties: edge.Properties,
			}
		}
	}
	return nil
}

type nodeResult struct {
	ID         string
	Kinds      []string
	Properties map[string]any
}

type edgeResult struct {
	Start      ein.EdgeEndpoint
	End        ein.EdgeEndpoint
	Kind       string
	Properties map[string]any
}
