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

package apiparser_test

import (
	"strings"
	"testing"

	"github.com/specterops/bloodhound/cmd/api/src/services/apiparser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testOpenAPI3Doc = `{
  "openapi": "3.0.1",
  "info": {
    "title": "Petstore API",
    "description": "A sample API for managing pets",
    "version": "1.0.0"
  },
  "servers": [
    {"url": "https://api.petstore.io/v1", "description": "Production"}
  ],
  "tags": [
    {"name": "pets", "description": "Pet operations"},
    {"name": "store", "description": "Store operations"}
  ],
  "paths": {
    "/pets": {
      "get": {
        "operationId": "listPets",
        "summary": "List all pets",
        "tags": ["pets"],
        "parameters": [
          {
            "name": "limit",
            "in": "query",
            "required": false,
            "description": "Max items to return",
            "schema": {"type": "integer", "format": "int32"}
          }
        ],
        "responses": {
          "200": {
            "description": "A list of pets",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/PetList"}
              }
            }
          }
        },
        "security": [{"api_key": []}]
      },
      "post": {
        "operationId": "createPet",
        "summary": "Create a pet",
        "tags": ["pets"],
        "requestBody": {
          "content": {
            "application/json": {
              "schema": {"$ref": "#/components/schemas/Pet"}
            }
          }
        },
        "responses": {
          "201": {
            "description": "Pet created",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/Pet"}
              }
            }
          }
        }
      }
    },
    "/pets/{petId}": {
      "parameters": [
        {"name": "petId", "in": "path", "required": true, "schema": {"type": "string"}}
      ],
      "get": {
        "operationId": "getPet",
        "summary": "Get a pet by ID",
        "tags": ["pets"],
        "responses": {
          "200": {
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/Pet"}
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "Pet": {
        "type": "object",
        "description": "A pet",
        "properties": {
          "id": {"type": "integer"},
          "name": {"type": "string"},
          "tag": {"type": "string"}
        }
      },
      "PetList": {
        "type": "array",
        "items": {"$ref": "#/components/schemas/Pet"}
      }
    },
    "securitySchemes": {
      "api_key": {
        "type": "apiKey",
        "in": "header",
        "description": "API key auth"
      }
    }
  }
}`

const testSwagger2Doc = `{
  "swagger": "2.0",
  "info": {
    "title": "Legacy API",
    "version": "2.0.0"
  },
  "host": "api.legacy.com",
  "basePath": "/v2",
  "schemes": ["https"],
  "tags": [
    {"name": "users"}
  ],
  "paths": {
    "/users": {
      "get": {
        "operationId": "listUsers",
        "tags": ["users"],
        "parameters": [
          {"name": "page", "in": "query", "type": "integer"}
        ],
        "responses": {
          "200": {
            "schema": {"$ref": "#/definitions/UserList"}
          }
        }
      }
    }
  },
  "definitions": {
    "User": {
      "type": "object",
      "properties": {
        "id": {"type": "integer"},
        "email": {"type": "string"}
      }
    },
    "UserList": {
      "type": "array",
      "items": {"$ref": "#/definitions/User"}
    }
  },
  "securityDefinitions": {
    "bearer": {
      "type": "http",
      "scheme": "bearer"
    }
  }
}`

func TestParse_OpenAPI3(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testOpenAPI3Doc))
	require.NoError(t, err)

	assert.Equal(t, "3.0.1", doc.SpecVersion)
	assert.Equal(t, "Petstore API", doc.Title)
	assert.Equal(t, "A sample API for managing pets", doc.Description)
	assert.Equal(t, "1.0.0", doc.Version)

	// Servers
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "https://api.petstore.io/v1", doc.Servers[0].URL)

	// Tags
	require.Len(t, doc.Tags, 2)
	assert.Equal(t, "pets", doc.Tags[0].Name)

	// Endpoints
	require.Len(t, doc.Endpoints, 3) // GET /pets, POST /pets, GET /pets/{petId}

	// Find the listPets endpoint
	var listPets apiparser.APIEndpoint
	for _, endpoint := range doc.Endpoints {
		if endpoint.OperationID == "listPets" {
			listPets = endpoint
			break
		}
	}
	assert.Equal(t, "/pets", listPets.Path)
	assert.Equal(t, "GET", listPets.Method)
	require.Len(t, listPets.Parameters, 1)
	assert.Equal(t, "limit", listPets.Parameters[0].Name)
	assert.Equal(t, "query", listPets.Parameters[0].In)
	assert.Equal(t, "integer", listPets.Parameters[0].Type)
	assert.Equal(t, "int32", listPets.Parameters[0].Format)

	// Response schema ref
	assert.Equal(t, "PetList", listPets.ResponseSchemaRefs["200"])

	// Security
	require.Len(t, listPets.SecurityRequirements, 1)
	assert.Equal(t, "api_key", listPets.SecurityRequirements[0])

	// Find createPet — should have request body schema ref
	var createPet apiparser.APIEndpoint
	for _, endpoint := range doc.Endpoints {
		if endpoint.OperationID == "createPet" {
			createPet = endpoint
			break
		}
	}
	assert.Equal(t, "Pet", createPet.RequestBodySchemaRef)

	// getPet should inherit path-level petId parameter
	var getPet apiparser.APIEndpoint
	for _, endpoint := range doc.Endpoints {
		if endpoint.OperationID == "getPet" {
			getPet = endpoint
			break
		}
	}
	require.Len(t, getPet.Parameters, 1)
	assert.Equal(t, "petId", getPet.Parameters[0].Name)
	assert.Equal(t, "path", getPet.Parameters[0].In)
	assert.True(t, getPet.Parameters[0].Required)

	// Schemas
	require.Len(t, doc.Schemas, 2)

	// Security schemes
	require.Len(t, doc.SecuritySchemes, 1)
	assert.Equal(t, "api_key", doc.SecuritySchemes[0].Name)
	assert.Equal(t, "apiKey", doc.SecuritySchemes[0].Type)
}

func TestParse_Swagger2(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testSwagger2Doc))
	require.NoError(t, err)

	assert.Equal(t, "2.0", doc.SpecVersion)
	assert.Equal(t, "Legacy API", doc.Title)
	assert.Equal(t, "2.0.0", doc.Version)

	// Server from host + basePath
	require.Len(t, doc.Servers, 1)
	assert.Equal(t, "https://api.legacy.com/v2", doc.Servers[0].URL)

	// Endpoints
	require.Len(t, doc.Endpoints, 1)
	assert.Equal(t, "listUsers", doc.Endpoints[0].OperationID)
	assert.Equal(t, "GET", doc.Endpoints[0].Method)

	// Schemas
	require.Len(t, doc.Schemas, 2)

	// Security schemes
	require.Len(t, doc.SecuritySchemes, 1)
	assert.Equal(t, "bearer", doc.SecuritySchemes[0].Name)
}

func TestParse_InvalidJSON(t *testing.T) {
	_, err := apiparser.Parse(strings.NewReader("not json"))
	assert.Error(t, err)
}

func TestParse_UnrecognisedSpec(t *testing.T) {
	_, err := apiparser.Parse(strings.NewReader(`{"foo":"bar"}`))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unrecognised API spec")
}

func TestConvert_ProducesExpectedGraph(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testOpenAPI3Doc))
	require.NoError(t, err)

	result := apiparser.Convert(doc, "petstore")

	// Count node kinds
	var (
		serviceCount  int
		endpointCount int
		paramCount    int
		schemaCount   int
		securityCount int
		tagCount      int
	)
	for _, node := range result.Nodes {
		for _, kind := range node.Kinds {
			switch kind {
			case apiparser.KindAPIService:
				serviceCount++
			case apiparser.KindAPIEndpoint:
				endpointCount++
			case apiparser.KindAPIParameter:
				paramCount++
			case apiparser.KindAPISchema:
				schemaCount++
			case apiparser.KindAPISecurityScheme:
				securityCount++
			case apiparser.KindAPITag:
				tagCount++
			}
		}
	}

	assert.Equal(t, 1, serviceCount, "should have 1 service node")
	assert.Equal(t, 3, endpointCount, "should have 3 endpoint nodes")
	// listPets has 1 param (limit), getPet has 1 param (petId), createPet has 0
	assert.Equal(t, 2, paramCount, "should have 2 parameter nodes")
	assert.Equal(t, 2, schemaCount, "should have 2 schema nodes")
	assert.Equal(t, 1, securityCount, "should have 1 security scheme node")
	assert.Equal(t, 2, tagCount, "should have 2 tag nodes")

	// Count edge kinds
	edgeCounts := make(map[string]int)
	for _, edge := range result.Edges {
		edgeCounts[edge.Kind]++
	}

	assert.Equal(t, 3, edgeCounts[apiparser.EdgeHasEndpoint], "3 HasEndpoint edges (service→endpoints)")
	assert.Equal(t, 2, edgeCounts[apiparser.EdgeHasParameter], "2 HasParameter edges")
	assert.Equal(t, 3, edgeCounts[apiparser.EdgeTaggedWith], "3 TaggedWith edges (all 3 endpoints tagged 'pets')")
	assert.Equal(t, 1, edgeCounts[apiparser.EdgeAcceptsSchema], "1 AcceptsSchema edge (createPet)")
	assert.GreaterOrEqual(t, edgeCounts[apiparser.EdgeReturnsSchema], 2, "at least 2 ReturnsSchema edges")
	assert.Equal(t, 1, edgeCounts[apiparser.EdgeRequiresSecurity], "1 RequiresSecurity edge (listPets)")
	assert.Equal(t, 1, edgeCounts[apiparser.EdgeReferencesSchema], "1 ReferencesSchema edge (PetList→Pet)")

	// Verify the service node has correct properties
	for _, node := range result.Nodes {
		for _, kind := range node.Kinds {
			if kind == apiparser.KindAPIService {
				assert.Equal(t, "Petstore API", node.Properties["name"])
				assert.Equal(t, "1.0.0", node.Properties["api_version"])
				break
			}
		}
	}
}

func TestConvert_DeterministicObjectIDs(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testOpenAPI3Doc))
	require.NoError(t, err)

	result1 := apiparser.Convert(doc, "petstore")
	result2 := apiparser.Convert(doc, "petstore")

	// Same input should produce same IDs
	require.Equal(t, len(result1.Nodes), len(result2.Nodes))
	for idx := range result1.Nodes {
		assert.Equal(t, result1.Nodes[idx].ID, result2.Nodes[idx].ID, "object IDs should be deterministic")
	}
}

func TestConvert_AllNodesHaveObjectID(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testOpenAPI3Doc))
	require.NoError(t, err)

	result := apiparser.Convert(doc, "petstore")

	for _, node := range result.Nodes {
		objectID, ok := node.Properties["objectid"]
		assert.True(t, ok, "node %s must have objectid property", node.ID)
		assert.Equal(t, node.ID, objectID, "objectid property must match node ID")
	}
}

func TestConvert_AllNodesHaveAPIBase(t *testing.T) {
	doc, err := apiparser.Parse(strings.NewReader(testOpenAPI3Doc))
	require.NoError(t, err)

	result := apiparser.Convert(doc, "petstore")

	for _, node := range result.Nodes {
		hasBase := false
		for _, kind := range node.Kinds {
			if kind == apiparser.KindAPIBase {
				hasBase = true
				break
			}
		}
		assert.True(t, hasBase, "node %s must have APIBase kind", node.ID)
	}
}
