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
	"github.com/specterops/bloodhound/cmd/api/src/model"
)

const (
	// ExtensionName is the name of the OpenGraph extension registered for API documentation kinds.
	ExtensionName = "apihound"

	// ExtensionDisplayName is the human-readable display name for the extension.
	ExtensionDisplayName = "APIHound"

	// ExtensionVersion is the version of the extension schema.
	ExtensionVersion = "1.0.0"

	// ExtensionNamespace is the namespace prefix used for the extension.
	// NOTE: The kind names in converter.go do not carry this prefix because
	// the extension is registered directly through the database layer,
	// bypassing service-layer namespace validation. This is intentional for
	// a fork-level integration where kinds are built-in rather than user-managed.
	ExtensionNamespace = "APIHound"
)

// BuildExtensionInput constructs the model.GraphExtensionInput that describes
// all API documentation node and relationship kinds. This input is used at
// startup to register (upsert) the APIHound OpenGraph extension so that the
// pathfinding and graph explorer UI recognise API edges and nodes.
func BuildExtensionInput() model.GraphExtensionInput {
	return model.GraphExtensionInput{
		ExtensionInput: model.ExtensionInput{
			Name:        ExtensionName,
			DisplayName: ExtensionDisplayName,
			Version:     ExtensionVersion,
			Namespace:   ExtensionNamespace,
		},

		NodeKindsInput: model.NodesInput{
			{Name: KindAPIBase, DisplayName: "API Base", Description: "Base kind shared by all API entities"},
			{Name: KindAPIService, DisplayName: "API Service", Description: "An API service described by an OpenAPI/Swagger document", IsDisplayKind: true},
			{Name: KindAPIEndpoint, DisplayName: "API Endpoint", Description: "A single API endpoint (path + HTTP method)", IsDisplayKind: true},
			{Name: KindAPIParameter, DisplayName: "API Parameter", Description: "A parameter accepted by an endpoint"},
			{Name: KindAPISchema, DisplayName: "API Schema", Description: "A data schema (request body or response)"},
			{Name: KindAPISecurityScheme, DisplayName: "API Security Scheme", Description: "An authentication / authorisation mechanism"},
			{Name: KindAPITag, DisplayName: "API Tag", Description: "A logical grouping tag"},
			{Name: KindAPIServer, DisplayName: "API Server", Description: "A server / base URL that hosts an API", IsDisplayKind: true},
			{Name: KindAPIOpCo, DisplayName: "API OpCo", Description: "An operational or business unit that owns API services", IsDisplayKind: true},
		},

		RelationshipKindsInput: model.RelationshipsInput{
			{Name: EdgeHasEndpoint, Description: "Service exposes this endpoint", IsTraversable: true},
			{Name: EdgeHasParameter, Description: "Endpoint accepts this parameter", IsTraversable: true},
			{Name: EdgeAcceptsSchema, Description: "Endpoint accepts this request schema", IsTraversable: true},
			{Name: EdgeReturnsSchema, Description: "Endpoint returns this response schema", IsTraversable: true},
			{Name: EdgeRequiresSecurity, Description: "Endpoint requires this security scheme", IsTraversable: true},
			{Name: EdgeTaggedWith, Description: "Endpoint is tagged with this tag", IsTraversable: true},
			{Name: EdgeReferencesSchema, Description: "Schema references another schema", IsTraversable: true},
			{Name: EdgeHostedOn, Description: "Service is hosted on this server", IsTraversable: true},
			{Name: EdgeCallsExternalAPI, Description: "Endpoint calls or consumes an external API server", IsTraversable: true},
			{Name: EdgeBelongsToOpCo, Description: "Service belongs to this operational/business unit", IsTraversable: true},
		},
	}
}
