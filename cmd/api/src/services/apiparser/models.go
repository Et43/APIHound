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

// APIDoc is the top-level representation of a parsed API specification,
// supporting both OpenAPI 3.x and Swagger 2.0 documents.
type APIDoc struct {
	// SpecVersion is the OpenAPI/Swagger spec version string (e.g. "3.0.1", "2.0").
	SpecVersion string

	// Title is the API title from the info block.
	Title string

	// Description is the API description from the info block.
	Description string

	// Version is the API version from the info block (distinct from spec version).
	Version string

	// Servers lists the base URLs / host information for the API.
	Servers []APIServer

	// Endpoints are the individual API operations extracted from paths.
	Endpoints []APIEndpoint

	// Schemas are the reusable model/schema definitions.
	Schemas []APISchema

	// SecuritySchemes are the authentication mechanisms defined by the API.
	SecuritySchemes []APISecurityScheme

	// Tags are logical groupings declared at document level.
	Tags []APITag
}

// APIServer represents a server / base URL for the API.
type APIServer struct {
	URL         string
	Description string
}

// APIEndpoint represents a single API operation (method + path combination).
type APIEndpoint struct {
	// Path is the URL path, e.g. "/users/{id}".
	Path string

	// Method is the HTTP method (GET, POST, etc.), uppercased.
	Method string

	// OperationID is the optional unique identifier for the operation.
	OperationID string

	// Summary is the short summary of the operation.
	Summary string

	// Description is the longer description.
	Description string

	// Tags are the tag names associated with this operation.
	Tags []string

	// Parameters are the parameters for this operation (path, query, header, cookie).
	Parameters []APIParameter

	// RequestBodySchemaRef is a reference to the request body schema, if any.
	RequestBodySchemaRef string

	// ResponseSchemaRefs maps status codes to their response schema references.
	ResponseSchemaRefs map[string]string

	// SecurityRequirements lists the security scheme names required for this endpoint.
	SecurityRequirements []string

	// Deprecated indicates whether this operation is marked as deprecated.
	Deprecated bool
}

// APIParameter represents a single parameter for an API endpoint.
type APIParameter struct {
	Name        string
	In          string // "query", "path", "header", "cookie"
	Required    bool
	Description string
	Type        string // simplified type: "string", "integer", "boolean", "array", "object"
	Format      string // e.g. "int64", "date-time"
}

// APISchema represents a reusable data model/schema definition.
type APISchema struct {
	Name        string
	Type        string // "object", "array", "string", etc.
	Description string
	Properties  []string // property names for objects
	References  []string // schema names referenced by this schema (via $ref, items, allOf, etc.)
}

// APISecurityScheme represents an authentication mechanism.
type APISecurityScheme struct {
	Name        string
	Type        string // "apiKey", "http", "oauth2", "openIdConnect"
	Description string
	In          string // for apiKey: "header", "query", "cookie"
	Scheme      string // for http: "bearer", "basic"
}

// APITag represents a logical grouping of operations.
type APITag struct {
	Name        string
	Description string
}
