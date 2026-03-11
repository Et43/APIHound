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
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// Parse reads an OpenAPI 3.x or Swagger 2.0 JSON document from the reader
// and returns a normalised APIDoc. The reader must provide valid JSON.
func Parse(reader io.Reader) (APIDoc, error) {
	var raw map[string]json.RawMessage
	if err := json.NewDecoder(reader).Decode(&raw); err != nil {
		return APIDoc{}, fmt.Errorf("failed to decode API spec JSON: %w", err)
	}

	if _, ok := raw["openapi"]; ok {
		return parseOpenAPI3(raw)
	}
	if _, ok := raw["swagger"]; ok {
		return parseSwagger2(raw)
	}
	return APIDoc{}, fmt.Errorf("unrecognised API spec: missing 'openapi' or 'swagger' key")
}

// ---------------------------------------------------------------------------
// OpenAPI 3.x
// ---------------------------------------------------------------------------

func parseOpenAPI3(raw map[string]json.RawMessage) (APIDoc, error) {
	var doc APIDoc

	// spec version
	if err := json.Unmarshal(raw["openapi"], &doc.SpecVersion); err != nil {
		return doc, fmt.Errorf("parsing openapi version: %w", err)
	}

	// info
	if infoRaw, ok := raw["info"]; ok {
		parseInfo(infoRaw, &doc)
	}

	// servers
	if serversRaw, ok := raw["servers"]; ok {
		doc.Servers = parseServers(serversRaw)
	}

	// tags
	if tagsRaw, ok := raw["tags"]; ok {
		doc.Tags = parseTags(tagsRaw)
	}

	// paths → endpoints
	if pathsRaw, ok := raw["paths"]; ok {
		doc.Endpoints = parsePathsV3(pathsRaw)
	}

	// components/schemas
	if componentsRaw, ok := raw["components"]; ok {
		var components map[string]json.RawMessage
		if err := json.Unmarshal(componentsRaw, &components); err == nil {
			if schemasRaw, ok := components["schemas"]; ok {
				doc.Schemas = parseSchemas(schemasRaw)
			}
			if securitySchemesRaw, ok := components["securitySchemes"]; ok {
				doc.SecuritySchemes = parseSecuritySchemes(securitySchemesRaw)
			}
		}
	}

	return doc, nil
}

// ---------------------------------------------------------------------------
// Swagger 2.0
// ---------------------------------------------------------------------------

func parseSwagger2(raw map[string]json.RawMessage) (APIDoc, error) {
	var doc APIDoc

	if err := json.Unmarshal(raw["swagger"], &doc.SpecVersion); err != nil {
		return doc, fmt.Errorf("parsing swagger version: %w", err)
	}

	// info
	if infoRaw, ok := raw["info"]; ok {
		parseInfo(infoRaw, &doc)
	}

	// host + basePath → single server
	var (
		host     string
		basePath string
		schemes  []string
	)
	if h, ok := raw["host"]; ok {
		json.Unmarshal(h, &host)
	}
	if b, ok := raw["basePath"]; ok {
		json.Unmarshal(b, &basePath)
	}
	if s, ok := raw["schemes"]; ok {
		json.Unmarshal(s, &schemes)
	}
	if host != "" {
		scheme := "https"
		if len(schemes) > 0 {
			scheme = schemes[0]
		}
		doc.Servers = []APIServer{{URL: scheme + "://" + host + basePath}}
	}

	// tags
	if tagsRaw, ok := raw["tags"]; ok {
		doc.Tags = parseTags(tagsRaw)
	}

	// paths → endpoints (Swagger 2 paths are structurally similar to OpenAPI 3)
	if pathsRaw, ok := raw["paths"]; ok {
		doc.Endpoints = parsePathsV2(pathsRaw)
	}

	// definitions → schemas
	if definitionsRaw, ok := raw["definitions"]; ok {
		doc.Schemas = parseSchemas(definitionsRaw)
	}

	// securityDefinitions → security schemes
	if secDefRaw, ok := raw["securityDefinitions"]; ok {
		doc.SecuritySchemes = parseSecuritySchemes(secDefRaw)
	}

	return doc, nil
}

// ---------------------------------------------------------------------------
// Shared parsing helpers
// ---------------------------------------------------------------------------

func parseInfo(data json.RawMessage, doc *APIDoc) {
	var info struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Version     string `json:"version"`
	}
	if err := json.Unmarshal(data, &info); err == nil {
		doc.Title = info.Title
		doc.Description = info.Description
		doc.Version = info.Version
	}
}

func parseServers(data json.RawMessage) []APIServer {
	var servers []struct {
		URL         string `json:"url"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil
	}
	var result []APIServer
	for _, server := range servers {
		result = append(result, APIServer{URL: server.URL, Description: server.Description})
	}
	return result
}

func parseTags(data json.RawMessage) []APITag {
	var tags []struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil
	}
	var result []APITag
	for _, tag := range tags {
		result = append(result, APITag{Name: tag.Name, Description: tag.Description})
	}
	return result
}

func parseSecuritySchemes(data json.RawMessage) []APISecurityScheme {
	var schemesMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &schemesMap); err != nil {
		return nil
	}
	var result []APISecurityScheme
	for name, schemeRaw := range schemesMap {
		var scheme struct {
			Type        string `json:"type"`
			Description string `json:"description"`
			In          string `json:"in"`
			Scheme      string `json:"scheme"`
		}
		if err := json.Unmarshal(schemeRaw, &scheme); err != nil {
			continue
		}
		result = append(result, APISecurityScheme{
			Name:        name,
			Type:        scheme.Type,
			Description: scheme.Description,
			In:          scheme.In,
			Scheme:      scheme.Scheme,
		})
	}
	return result
}

func parseSchemas(data json.RawMessage) []APISchema {
	var schemasMap map[string]json.RawMessage
	if err := json.Unmarshal(data, &schemasMap); err != nil {
		return nil
	}
	var result []APISchema
	for name, schemaRaw := range schemasMap {
		result = append(result, parseSingleSchema(name, schemaRaw))
	}
	return result
}

func parseSingleSchema(name string, data json.RawMessage) APISchema {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return APISchema{Name: name}
	}

	schema := APISchema{Name: name}

	if typeRaw, ok := raw["type"]; ok {
		json.Unmarshal(typeRaw, &schema.Type)
	}
	if descRaw, ok := raw["description"]; ok {
		json.Unmarshal(descRaw, &schema.Description)
	}

	// Collect property names
	if propsRaw, ok := raw["properties"]; ok {
		var propsMap map[string]json.RawMessage
		if err := json.Unmarshal(propsRaw, &propsMap); err == nil {
			for propName, propRaw := range propsMap {
				schema.Properties = append(schema.Properties, propName)
				// Check for $ref inside each property
				schema.References = append(schema.References, extractRefs(propRaw)...)
			}
		}
	}

	// allOf, anyOf, oneOf — collect references
	for _, compositionKey := range []string{"allOf", "anyOf", "oneOf"} {
		if compRaw, ok := raw[compositionKey]; ok {
			var items []json.RawMessage
			if err := json.Unmarshal(compRaw, &items); err == nil {
				for _, item := range items {
					schema.References = append(schema.References, extractRefs(item)...)
				}
			}
		}
	}

	// items (for arrays)
	if itemsRaw, ok := raw["items"]; ok {
		schema.References = append(schema.References, extractRefs(itemsRaw)...)
	}

	// Direct $ref at schema level
	schema.References = append(schema.References, extractRefs(data)...)

	// Deduplicate references
	schema.References = dedup(schema.References)

	return schema
}

// extractRefs looks for "$ref" keys in a JSON object and returns the referenced
// schema names. References look like "#/components/schemas/Foo" or "#/definitions/Bar".
func extractRefs(data json.RawMessage) []string {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return nil
	}

	var refs []string
	if refRaw, ok := obj["$ref"]; ok {
		var refStr string
		if err := json.Unmarshal(refRaw, &refStr); err == nil {
			if name := refToSchemaName(refStr); name != "" {
				refs = append(refs, name)
			}
		}
	}

	// Recurse into items, allOf, anyOf, oneOf
	if itemsRaw, ok := obj["items"]; ok {
		refs = append(refs, extractRefs(itemsRaw)...)
	}
	for _, key := range []string{"allOf", "anyOf", "oneOf"} {
		if arrRaw, ok := obj[key]; ok {
			var items []json.RawMessage
			if err := json.Unmarshal(arrRaw, &items); err == nil {
				for _, item := range items {
					refs = append(refs, extractRefs(item)...)
				}
			}
		}
	}

	return refs
}

// refToSchemaName extracts the schema name from a $ref string like
// "#/components/schemas/Foo" or "#/definitions/Bar".
func refToSchemaName(ref string) string {
	parts := strings.Split(ref, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// ---------------------------------------------------------------------------
// Path parsing — OpenAPI 3.x
// ---------------------------------------------------------------------------

var httpMethods = []string{"get", "post", "put", "delete", "patch", "head", "options", "trace"}

func parsePathsV3(data json.RawMessage) []APIEndpoint {
	var paths map[string]json.RawMessage
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil
	}

	var endpoints []APIEndpoint
	for path, pathItemRaw := range paths {
		var pathItem map[string]json.RawMessage
		if err := json.Unmarshal(pathItemRaw, &pathItem); err != nil {
			continue
		}

		// Path-level parameters apply to all operations under this path.
		var pathParams []APIParameter
		if paramsRaw, ok := pathItem["parameters"]; ok {
			pathParams = parseParameters(paramsRaw)
		}

		for _, method := range httpMethods {
			opRaw, ok := pathItem[method]
			if !ok {
				continue
			}
			endpoint := parseOperationV3(path, method, opRaw)
			// Merge path-level parameters (operation params override path params by name+in)
			endpoint.Parameters = mergeParameters(pathParams, endpoint.Parameters)
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func parseOperationV3(path, method string, data json.RawMessage) APIEndpoint {
	var op map[string]json.RawMessage
	if err := json.Unmarshal(data, &op); err != nil {
		return APIEndpoint{Path: path, Method: strings.ToUpper(method)}
	}

	endpoint := APIEndpoint{
		Path:   path,
		Method: strings.ToUpper(method),
	}

	if v, ok := op["operationId"]; ok {
		json.Unmarshal(v, &endpoint.OperationID)
	}
	if v, ok := op["summary"]; ok {
		json.Unmarshal(v, &endpoint.Summary)
	}
	if v, ok := op["description"]; ok {
		json.Unmarshal(v, &endpoint.Description)
	}
	if v, ok := op["tags"]; ok {
		json.Unmarshal(v, &endpoint.Tags)
	}
	if v, ok := op["deprecated"]; ok {
		json.Unmarshal(v, &endpoint.Deprecated)
	}

	// parameters
	if paramsRaw, ok := op["parameters"]; ok {
		endpoint.Parameters = parseParameters(paramsRaw)
	}

	// requestBody
	if reqBodyRaw, ok := op["requestBody"]; ok {
		endpoint.RequestBodySchemaRef = extractRequestBodySchemaRef(reqBodyRaw)
	}

	// responses
	endpoint.ResponseSchemaRefs = make(map[string]string)
	if responsesRaw, ok := op["responses"]; ok {
		var responses map[string]json.RawMessage
		if err := json.Unmarshal(responsesRaw, &responses); err == nil {
			for statusCode, respRaw := range responses {
				if ref := extractResponseSchemaRef(respRaw); ref != "" {
					endpoint.ResponseSchemaRefs[statusCode] = ref
				}
			}
		}
	}

	// security
	if secRaw, ok := op["security"]; ok {
		endpoint.SecurityRequirements = parseSecurityRequirements(secRaw)
	}

	return endpoint
}

func extractRequestBodySchemaRef(data json.RawMessage) string {
	var body map[string]json.RawMessage
	if err := json.Unmarshal(data, &body); err != nil {
		return ""
	}
	contentRaw, ok := body["content"]
	if !ok {
		return ""
	}
	var content map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &content); err != nil {
		return ""
	}
	// Try common content types
	for _, contentType := range []string{"application/json", "application/xml", "*/*"} {
		if mediaRaw, ok := content[contentType]; ok {
			var media map[string]json.RawMessage
			if err := json.Unmarshal(mediaRaw, &media); err == nil {
				if schemaRaw, ok := media["schema"]; ok {
					refs := extractRefs(schemaRaw)
					if len(refs) > 0 {
						return refs[0]
					}
				}
			}
		}
	}
	return ""
}

func extractResponseSchemaRef(data json.RawMessage) string {
	var resp map[string]json.RawMessage
	if err := json.Unmarshal(data, &resp); err != nil {
		return ""
	}
	contentRaw, ok := resp["content"]
	if !ok {
		// Swagger 2.0 style: schema directly on the response
		if schemaRaw, ok := resp["schema"]; ok {
			refs := extractRefs(schemaRaw)
			if len(refs) > 0 {
				return refs[0]
			}
		}
		return ""
	}
	var content map[string]json.RawMessage
	if err := json.Unmarshal(contentRaw, &content); err != nil {
		return ""
	}
	for _, contentType := range []string{"application/json", "application/xml", "*/*"} {
		if mediaRaw, ok := content[contentType]; ok {
			var media map[string]json.RawMessage
			if err := json.Unmarshal(mediaRaw, &media); err == nil {
				if schemaRaw, ok := media["schema"]; ok {
					refs := extractRefs(schemaRaw)
					if len(refs) > 0 {
						return refs[0]
					}
				}
			}
		}
	}
	return ""
}

func parseParameters(data json.RawMessage) []APIParameter {
	var params []struct {
		Name        string          `json:"name"`
		In          string          `json:"in"`
		Required    bool            `json:"required"`
		Description string          `json:"description"`
		Schema      json.RawMessage `json:"schema"`
		Type        string          `json:"type"`   // Swagger 2.0
		Format      string          `json:"format"` // Swagger 2.0
	}
	if err := json.Unmarshal(data, &params); err != nil {
		return nil
	}

	var result []APIParameter
	for _, param := range params {
		apiParam := APIParameter{
			Name:        param.Name,
			In:          param.In,
			Required:    param.Required,
			Description: param.Description,
			Type:        param.Type,
			Format:      param.Format,
		}
		// OpenAPI 3.x: type is inside the schema object
		if param.Schema != nil && apiParam.Type == "" {
			var s struct {
				Type   string `json:"type"`
				Format string `json:"format"`
			}
			if err := json.Unmarshal(param.Schema, &s); err == nil {
				apiParam.Type = s.Type
				if s.Format != "" {
					apiParam.Format = s.Format
				}
			}
		}
		result = append(result, apiParam)
	}
	return result
}

func parseSecurityRequirements(data json.RawMessage) []string {
	var requirements []map[string]json.RawMessage
	if err := json.Unmarshal(data, &requirements); err != nil {
		return nil
	}
	var names []string
	for _, req := range requirements {
		for name := range req {
			names = append(names, name)
		}
	}
	return dedup(names)
}

// ---------------------------------------------------------------------------
// Path parsing — Swagger 2.0
// ---------------------------------------------------------------------------

func parsePathsV2(data json.RawMessage) []APIEndpoint {
	var paths map[string]json.RawMessage
	if err := json.Unmarshal(data, &paths); err != nil {
		return nil
	}

	var endpoints []APIEndpoint
	for path, pathItemRaw := range paths {
		var pathItem map[string]json.RawMessage
		if err := json.Unmarshal(pathItemRaw, &pathItem); err != nil {
			continue
		}

		// Path-level parameters
		var pathParams []APIParameter
		if paramsRaw, ok := pathItem["parameters"]; ok {
			pathParams = parseParameters(paramsRaw)
		}

		for _, method := range httpMethods {
			opRaw, ok := pathItem[method]
			if !ok {
				continue
			}
			endpoint := parseOperationV2(path, method, opRaw)
			endpoint.Parameters = mergeParameters(pathParams, endpoint.Parameters)
			endpoints = append(endpoints, endpoint)
		}
	}
	return endpoints
}

func parseOperationV2(path, method string, data json.RawMessage) APIEndpoint {
	var op map[string]json.RawMessage
	if err := json.Unmarshal(data, &op); err != nil {
		return APIEndpoint{Path: path, Method: strings.ToUpper(method)}
	}

	endpoint := APIEndpoint{
		Path:   path,
		Method: strings.ToUpper(method),
	}

	if v, ok := op["operationId"]; ok {
		json.Unmarshal(v, &endpoint.OperationID)
	}
	if v, ok := op["summary"]; ok {
		json.Unmarshal(v, &endpoint.Summary)
	}
	if v, ok := op["description"]; ok {
		json.Unmarshal(v, &endpoint.Description)
	}
	if v, ok := op["tags"]; ok {
		json.Unmarshal(v, &endpoint.Tags)
	}
	if v, ok := op["deprecated"]; ok {
		json.Unmarshal(v, &endpoint.Deprecated)
	}

	// parameters (in Swagger 2.0, body params carry the schema)
	if paramsRaw, ok := op["parameters"]; ok {
		var rawParams []map[string]json.RawMessage
		if err := json.Unmarshal(paramsRaw, &rawParams); err == nil {
			for _, rawParam := range rawParams {
				var inVal string
				if inRaw, ok := rawParam["in"]; ok {
					json.Unmarshal(inRaw, &inVal)
				}
				if inVal == "body" {
					// Body parameter: extract schema ref
					if schemaRaw, ok := rawParam["schema"]; ok {
						refs := extractRefs(schemaRaw)
						if len(refs) > 0 {
							endpoint.RequestBodySchemaRef = refs[0]
						}
					}
				}
			}
		}
		endpoint.Parameters = parseParameters(paramsRaw)
	}

	// responses
	endpoint.ResponseSchemaRefs = make(map[string]string)
	if responsesRaw, ok := op["responses"]; ok {
		var responses map[string]json.RawMessage
		if err := json.Unmarshal(responsesRaw, &responses); err == nil {
			for statusCode, respRaw := range responses {
				if ref := extractResponseSchemaRef(respRaw); ref != "" {
					endpoint.ResponseSchemaRefs[statusCode] = ref
				}
			}
		}
	}

	// security
	if secRaw, ok := op["security"]; ok {
		endpoint.SecurityRequirements = parseSecurityRequirements(secRaw)
	}

	return endpoint
}

// ---------------------------------------------------------------------------
// Utilities
// ---------------------------------------------------------------------------

// mergeParameters merges path-level parameters with operation-level parameters.
// Operation params override path params when both share the same name+in combination.
func mergeParameters(pathParams, opParams []APIParameter) []APIParameter {
	if len(pathParams) == 0 {
		return opParams
	}
	if len(opParams) == 0 {
		return pathParams
	}

	// Build a set of keys that the operation already defines.
	opKeys := make(map[string]struct{}, len(opParams))
	for _, param := range opParams {
		opKeys[param.Name+"|"+param.In] = struct{}{}
	}

	merged := make([]APIParameter, 0, len(pathParams)+len(opParams))
	for _, param := range pathParams {
		if _, overridden := opKeys[param.Name+"|"+param.In]; !overridden {
			merged = append(merged, param)
		}
	}
	merged = append(merged, opParams...)
	return merged
}

func dedup(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	var result []string
	for _, item := range items {
		if _, ok := seen[item]; !ok {
			seen[item] = struct{}{}
			result = append(result, item)
		}
	}
	return result
}
