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

import { CommonSearchType } from './types';

const categoryAPI = 'API';

export const CommonSearches: CommonSearchType[] = [
    {
        subheader: 'API Discovery',
        category: categoryAPI,
        queries: [
            {
                name: 'All API Services',
                description: 'List every API service that has been ingested',
                query: `MATCH (s:APIService)\nRETURN s\nLIMIT 1000`,
            },
            {
                name: 'All API Endpoints',
                description: 'List every API endpoint across all services',
                query: `MATCH (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nRETURN s, e\nLIMIT 1000`,
            },
            {
                name: 'All API Endpoints for a given Service',
                description: 'Shows all endpoints belonging to a specific API service (match by name)',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE s.name =~ '(?i).*SERVICE.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'All API Servers (hosts)',
                description: 'List every server/host that APIs are hosted on',
                query: `MATCH (srv:APIServer)\nRETURN srv\nLIMIT 1000`,
            },
            {
                name: 'All API Services hosted on a given Server',
                description: 'Shows all services hosted on a specific server URL',
                query: `MATCH p = (s:APIService)-[:HostedOn]->(srv:APIServer)\nWHERE srv.url =~ '(?i).*HOST.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'All API Endpoints for a given Server',
                description: 'Shows all endpoints for services hosted on a specific server',
                query: `MATCH p = (srv:APIServer)<-[:HostedOn]-(s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE srv.url =~ '(?i).*HOST.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Map API Service to Server topology',
                description: 'Shows the relationship between all API services and their servers',
                query: `MATCH p = (s:APIService)-[:HostedOn]->(srv:APIServer)\nRETURN p\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'Authentication & Security',
        category: categoryAPI,
        queries: [
            {
                name: 'All API Security Schemes',
                description: 'List every authentication/authorization scheme across all services',
                query: `MATCH (sec:APISecurityScheme)\nRETURN sec\nLIMIT 1000`,
            },
            {
                name: 'Authentication schemes for a given Service',
                description: 'Shows all security schemes required by endpoints of a specific service',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:RequiresSecurity]->(sec:APISecurityScheme)\nWHERE s.name =~ '(?i).*SERVICE.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoints using OAuth2 authentication',
                description: 'Find all endpoints that require OAuth2 security',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:RequiresSecurity]->(sec:APISecurityScheme)\nWHERE sec.type = 'oauth2'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoints using API Key authentication',
                description: 'Find all endpoints that require API Key security',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:RequiresSecurity]->(sec:APISecurityScheme)\nWHERE sec.type = 'apiKey'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoints using HTTP Basic/Bearer authentication',
                description: 'Find all endpoints that use HTTP authentication (basic, bearer, etc.)',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:RequiresSecurity]->(sec:APISecurityScheme)\nWHERE sec.type = 'http'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoints with no authentication required',
                description: 'Find endpoints that do not require any security scheme',
                query: `MATCH (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE NOT (e)-[:RequiresSecurity]->(:APISecurityScheme)\nRETURN s, e\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'Endpoint Analysis',
        category: categoryAPI,
        queries: [
            {
                name: 'All deprecated endpoints',
                description: 'Find API endpoints marked as deprecated',
                query: `MATCH (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE e.deprecated = true\nRETURN s, e\nLIMIT 1000`,
            },
            {
                name: 'Endpoints by HTTP method',
                description: 'Find all endpoints using a specific HTTP method (replace METHOD with GET, POST, PUT, DELETE, etc.)',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE e.method = 'METHOD'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'All POST endpoints',
                description: 'Find all endpoints that accept POST requests (data creation)',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE e.method = 'POST'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'All DELETE endpoints',
                description: 'Find all endpoints that accept DELETE requests (destructive operations)',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nWHERE e.method = 'DELETE'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoint parameters',
                description: 'Shows all parameters for a specific endpoint path',
                query: `MATCH p = (e:APIEndpoint)-[:HasParameter]->(param:APIParameter)\nWHERE e.path =~ '(?i).*PATH.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Required parameters across all endpoints',
                description: 'Find all required parameters across all endpoints',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:HasParameter]->(param:APIParameter)\nWHERE param.required = true\nRETURN p\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'Schema & Data Models',
        category: categoryAPI,
        queries: [
            {
                name: 'All API Schemas',
                description: 'List every data schema/model defined across all services',
                query: `MATCH (schema:APISchema)\nWHERE NOT COALESCE(schema.external, false)\nRETURN schema\nLIMIT 1000`,
            },
            {
                name: 'Schemas for a given Service',
                description: 'Shows all schemas used by endpoints of a specific service',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:AcceptsSchema|ReturnsSchema]->(schema:APISchema)\nWHERE s.name =~ '(?i).*SERVICE.*'\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Schema reference graph',
                description: 'Shows how schemas reference each other (composition, inheritance)',
                query: `MATCH p = (a:APISchema)-[:ReferencesSchema]->(b:APISchema)\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Request body schemas',
                description: 'Shows all schemas used as request bodies',
                query: `MATCH p = (e:APIEndpoint)-[:AcceptsSchema]->(schema:APISchema)\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Response schemas',
                description: 'Shows all schemas used as responses',
                query: `MATCH p = (e:APIEndpoint)-[:ReturnsSchema]->(schema:APISchema)\nRETURN p\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'Tags & Organization',
        category: categoryAPI,
        queries: [
            {
                name: 'All API Tags',
                description: 'List every tag used to organize API endpoints',
                query: `MATCH (tag:APITag)\nRETURN tag\nLIMIT 1000`,
            },
            {
                name: 'Endpoints grouped by tag',
                description: 'Shows all endpoints and their associated tags',
                query: `MATCH p = (e:APIEndpoint)-[:TaggedWith]->(tag:APITag)\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'Endpoints for a specific tag',
                description: 'Shows all endpoints tagged with a specific tag name',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:TaggedWith]->(tag:APITag)\nWHERE tag.name =~ '(?i).*TAG.*'\nRETURN p\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'External Dependencies',
        category: categoryAPI,
        queries: [
            {
                name: 'Endpoints calling external APIs',
                description: 'Find endpoints that reference or call external API servers',
                query: `MATCH p = (e:APIEndpoint)-[:CallsExternalAPI]->(srv:APIServer)\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'External API dependency map',
                description: 'Shows the full chain from service to endpoint to external server',
                query: `MATCH p = (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)-[:CallsExternalAPI]->(ext:APIServer)\nRETURN p\nLIMIT 1000`,
            },
            {
                name: 'External schemas (unresolved references)',
                description: 'Find schemas that are referenced but not defined in the current spec',
                query: `MATCH (schema:APISchema)\nWHERE schema.external = true\nRETURN schema\nLIMIT 1000`,
            },
        ],
    },
    {
        subheader: 'Full API Surface',
        category: categoryAPI,
        queries: [
            {
                name: 'Full API graph for a Service',
                description: 'Shows the complete graph of a service: endpoints, parameters, schemas, security, tags, and servers',
                query: `MATCH (s:APIService)\nWHERE s.name =~ '(?i).*SERVICE.*'\nOPTIONAL MATCH p1 = (s)-[:HasEndpoint]->(e:APIEndpoint)\nOPTIONAL MATCH p2 = (e)-[:HasParameter]->(param:APIParameter)\nOPTIONAL MATCH p3 = (e)-[:AcceptsSchema|ReturnsSchema]->(schema:APISchema)\nOPTIONAL MATCH p4 = (e)-[:RequiresSecurity]->(sec:APISecurityScheme)\nOPTIONAL MATCH p5 = (e)-[:TaggedWith]->(tag:APITag)\nOPTIONAL MATCH p6 = (s)-[:HostedOn]->(srv:APIServer)\nRETURN s, p1, p2, p3, p4, p5, p6\nLIMIT 1000`,
            },
            {
                name: 'Services with most endpoints',
                description: 'Rank API services by number of endpoints',
                query: `MATCH (s:APIService)-[:HasEndpoint]->(e:APIEndpoint)\nRETURN s.name, COUNT(e) AS endpoint_count\nORDER BY endpoint_count DESC\nLIMIT 25`,
            },
        ],
    },
];
