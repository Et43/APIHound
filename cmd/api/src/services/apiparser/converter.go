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
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/specterops/bloodhound/packages/go/ein"
)

// Graph kind constants for API documentation entities. These are the string
// labels applied to nodes and edges in the dawgs graph database.
const (
	// Node kinds
	KindAPIBase           = "APIBase"
	KindAPIService        = "APIService"
	KindAPIEndpoint       = "APIEndpoint"
	KindAPIParameter      = "APIParameter"
	KindAPISchema         = "APISchema"
	KindAPISecurityScheme = "APISecurityScheme"
	KindAPITag            = "APITag"
	KindAPIServer         = "APIServer"

	// Edge kinds
	EdgeHasEndpoint       = "HasEndpoint"
	EdgeHasParameter      = "HasParameter"
	EdgeAcceptsSchema     = "AcceptsSchema"
	EdgeReturnsSchema     = "ReturnsSchema"
	EdgeRequiresSecurity  = "RequiresSecurity"
	EdgeTaggedWith        = "TaggedWith"
	EdgeReferencesSchema  = "ReferencesSchema"
	EdgeHostedOn          = "HostedOn"
	EdgeCallsExternalAPI  = "CallsExternalAPI"
)

// ConvertResult holds the generic nodes and edges produced from an API document,
// ready for ingestion through the OpenGraph pipeline.
type ConvertResult struct {
	Nodes []ein.GenericNode
	Edges []ein.GenericEdge
}

// Convert transforms a parsed APIDoc into graph nodes and edges.
// The serviceID is a caller-provided stable identifier for the API
// (typically derived from the uploaded filename or an explicit label).
func Convert(doc APIDoc, serviceID string) ConvertResult {
	var result ConvertResult

	serviceObjectID := objectID("service", serviceID)

	// Service node — the root of the API document
	// NOTE: the first kind becomes the primary_kind property in the graph,
	// which determines the icon the UI renders. Specific kind must come first.
	result.Nodes = append(result.Nodes, ein.GenericNode{
		ID:    serviceObjectID,
		Kinds: []string{KindAPIService, KindAPIBase},
		Properties: map[string]any{
			"objectid":     serviceObjectID,
			"name":         doc.Title,
			"description":  doc.Description,
			"api_version":  doc.Version,
			"spec_version": doc.SpecVersion,
		},
	})

	// Servers — objectIDs are based purely on normalised URL so the same
	// server from different specs converges to a single node in the graph.
	for _, server := range doc.Servers {
		normalizedURL := normalizeServerURL(server.URL)
		serverOID := objectID("server-url", normalizedURL)

		result.Nodes = append(result.Nodes, ein.GenericNode{
			ID:    serverOID,
			Kinds: []string{KindAPIServer, KindAPIBase},
			Properties: map[string]any{
				"objectid":    serverOID,
				"name":        server.URL,
				"url":         server.URL,
				"description": server.Description,
			},
		})

		// Service → Server edge
		result.Edges = append(result.Edges, ein.GenericEdge{
			Start:      ein.EdgeEndpoint{Value: serviceObjectID},
			End:        ein.EdgeEndpoint{Value: serverOID},
			Kind:       EdgeHostedOn,
			Properties: map[string]any{},
		})
	}

	// Tags
	tagObjectIDs := make(map[string]string) // tag name → objectID
	for _, tag := range doc.Tags {
		tagOID := objectID("tag", serviceID, tag.Name)
		tagObjectIDs[tag.Name] = tagOID
		result.Nodes = append(result.Nodes, ein.GenericNode{
			ID:    tagOID,
			Kinds: []string{KindAPITag, KindAPIBase},
			Properties: map[string]any{
				"objectid":    tagOID,
				"name":        tag.Name,
				"description": tag.Description,
			},
		})
	}

	// Security schemes
	secSchemeObjectIDs := make(map[string]string) // scheme name → objectID
	for _, scheme := range doc.SecuritySchemes {
		schemeOID := objectID("security", serviceID, scheme.Name)
		secSchemeObjectIDs[scheme.Name] = schemeOID
		result.Nodes = append(result.Nodes, ein.GenericNode{
			ID:    schemeOID,
			Kinds: []string{KindAPISecurityScheme, KindAPIBase},
			Properties: map[string]any{
				"objectid":      schemeOID,
				"name":          scheme.Name,
				"type":          scheme.Type,
				"description":   scheme.Description,
				"in":            scheme.In,
				"scheme":        scheme.Scheme,
			},
		})
	}

	// Schemas
	schemaObjectIDs := make(map[string]string) // schema name → objectID
	for _, schema := range doc.Schemas {
		schemaOID := objectID("schema", serviceID, schema.Name)
		schemaObjectIDs[schema.Name] = schemaOID

		properties := map[string]any{
			"objectid":    schemaOID,
			"name":        schema.Name,
			"type":        schema.Type,
			"description": schema.Description,
		}
		if len(schema.Properties) > 0 {
			properties["schema_properties"] = strings.Join(schema.Properties, ", ")
		}

		result.Nodes = append(result.Nodes, ein.GenericNode{
			ID:         schemaOID,
			Kinds:      []string{KindAPISchema, KindAPIBase},
			Properties: properties,
		})
	}

	// Schema → Schema edges (references)
	for _, schema := range doc.Schemas {
		sourceOID := schemaObjectIDs[schema.Name]
		for _, refName := range schema.References {
			targetOID, ok := schemaObjectIDs[refName]
			if !ok {
				// Referenced schema not in this document — create a placeholder node
				targetOID = objectID("schema", serviceID, refName)
				schemaObjectIDs[refName] = targetOID
				result.Nodes = append(result.Nodes, ein.GenericNode{
					ID:    targetOID,
					Kinds: []string{KindAPISchema, KindAPIBase},
					Properties: map[string]any{
						"objectid": targetOID,
						"name":     refName,
						"external": true,
					},
				})
			}
			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: sourceOID},
				End:        ein.EdgeEndpoint{Value: targetOID},
				Kind:       EdgeReferencesSchema,
				Properties: map[string]any{},
			})
		}
	}

	// Endpoints
	for _, endpoint := range doc.Endpoints {
		endpointOID := objectID("endpoint", serviceID, endpoint.Method, endpoint.Path)

		endpointName := endpoint.Method + " " + endpoint.Path
		if endpoint.OperationID != "" {
			endpointName = endpoint.OperationID + " (" + endpoint.Method + " " + endpoint.Path + ")"
		}

		endpointProperties := map[string]any{
			"objectid":     endpointOID,
			"name":         endpointName,
			"path":         endpoint.Path,
			"method":       endpoint.Method,
			"operation_id": endpoint.OperationID,
			"summary":      endpoint.Summary,
			"description":  endpoint.Description,
			"deprecated":   endpoint.Deprecated,
		}

		result.Nodes = append(result.Nodes, ein.GenericNode{
			ID:         endpointOID,
			Kinds:      []string{KindAPIEndpoint, KindAPIBase},
			Properties: endpointProperties,
		})

		// Service → Endpoint edge
		result.Edges = append(result.Edges, ein.GenericEdge{
			Start:      ein.EdgeEndpoint{Value: serviceObjectID},
			End:        ein.EdgeEndpoint{Value: endpointOID},
			Kind:       EdgeHasEndpoint,
			Properties: map[string]any{},
		})

		// Endpoint → Tag edges
		for _, tagName := range endpoint.Tags {
			tagOID, ok := tagObjectIDs[tagName]
			if !ok {
				// Tag referenced but not declared at doc level — create it
				tagOID = objectID("tag", serviceID, tagName)
				tagObjectIDs[tagName] = tagOID
				result.Nodes = append(result.Nodes, ein.GenericNode{
					ID:    tagOID,
					Kinds: []string{KindAPITag, KindAPIBase},
					Properties: map[string]any{
						"objectid": tagOID,
						"name":     tagName,
					},
				})
			}
			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: tagOID},
				Kind:       EdgeTaggedWith,
				Properties: map[string]any{},
			})
		}

		// Parameters
		for _, param := range endpoint.Parameters {
			paramOID := objectID("param", serviceID, endpoint.Method, endpoint.Path, param.Name, param.In)

			result.Nodes = append(result.Nodes, ein.GenericNode{
				ID:    paramOID,
				Kinds: []string{KindAPIParameter, KindAPIBase},
				Properties: map[string]any{
					"objectid":    paramOID,
					"name":        param.Name,
					"in":          param.In,
					"required":    param.Required,
					"description": param.Description,
					"type":        param.Type,
					"format":      param.Format,
				},
			})

			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: paramOID},
				Kind:       EdgeHasParameter,
				Properties: map[string]any{},
			})
		}

		// Request body schema reference
		if endpoint.RequestBodySchemaRef != "" {
			schemaOID, ok := schemaObjectIDs[endpoint.RequestBodySchemaRef]
			if !ok {
				schemaOID = objectID("schema", serviceID, endpoint.RequestBodySchemaRef)
				schemaObjectIDs[endpoint.RequestBodySchemaRef] = schemaOID
				result.Nodes = append(result.Nodes, ein.GenericNode{
					ID:    schemaOID,
					Kinds: []string{KindAPISchema, KindAPIBase},
					Properties: map[string]any{
						"objectid": schemaOID,
						"name":     endpoint.RequestBodySchemaRef,
						"external": true,
					},
				})
			}
			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: schemaOID},
				Kind:       EdgeAcceptsSchema,
				Properties: map[string]any{},
			})
		}

		// Response schema references
		for statusCode, schemaRef := range endpoint.ResponseSchemaRefs {
			schemaOID, ok := schemaObjectIDs[schemaRef]
			if !ok {
				schemaOID = objectID("schema", serviceID, schemaRef)
				schemaObjectIDs[schemaRef] = schemaOID
				result.Nodes = append(result.Nodes, ein.GenericNode{
					ID:    schemaOID,
					Kinds: []string{KindAPISchema, KindAPIBase},
					Properties: map[string]any{
						"objectid": schemaOID,
						"name":     schemaRef,
						"external": true,
					},
				})
			}
			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: schemaOID},
				Kind:       EdgeReturnsSchema,
				Properties: map[string]any{"status_code": statusCode},
			})
		}

		// External API reference (cross-service link)
		if endpoint.ExternalDocsURL != "" {
			normalizedURL := normalizeServerURL(endpoint.ExternalDocsURL)
			externalServerOID := objectID("server-url", normalizedURL)

			// Create the server node if it doesn't exist yet — this is
			// safe because graph upserts by objectid, so a duplicate
			// create is a no-op.
			result.Nodes = append(result.Nodes, ein.GenericNode{
				ID:    externalServerOID,
				Kinds: []string{KindAPIServer, KindAPIBase},
				Properties: map[string]any{
					"objectid": externalServerOID,
					"name":     endpoint.ExternalDocsURL,
					"url":      endpoint.ExternalDocsURL,
				},
			})

			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: externalServerOID},
				Kind:       EdgeCallsExternalAPI,
				Properties: map[string]any{},
			})
		}

		// Security requirements
		for _, schemeName := range endpoint.SecurityRequirements {
			schemeOID, ok := secSchemeObjectIDs[schemeName]
			if !ok {
				schemeOID = objectID("security", serviceID, schemeName)
				secSchemeObjectIDs[schemeName] = schemeOID
				result.Nodes = append(result.Nodes, ein.GenericNode{
					ID:    schemeOID,
					Kinds: []string{KindAPISecurityScheme, KindAPIBase},
					Properties: map[string]any{
						"objectid": schemeOID,
						"name":     schemeName,
						"external": true,
					},
				})
			}
			result.Edges = append(result.Edges, ein.GenericEdge{
				Start:      ein.EdgeEndpoint{Value: endpointOID},
				End:        ein.EdgeEndpoint{Value: schemeOID},
				Kind:       EdgeRequiresSecurity,
				Properties: map[string]any{},
			})
		}
	}

	return result
}

// objectID produces a deterministic, uppercase hex-string identifier from the
// given parts. Using SHA-256 keeps IDs short and collision-resistant.
func objectID(parts ...string) string {
	hash := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return strings.ToUpper(fmt.Sprintf("%x", hash[:16]))
}

// normalizeServerURL strips trailing slashes and lowercases the scheme+host
// so that "https://api.example.com/" and "https://api.example.com" produce
// the same objectID.
func normalizeServerURL(rawURL string) string {
	normalized := strings.TrimRight(rawURL, "/")
	normalized = strings.ToLower(normalized)
	return normalized
}
