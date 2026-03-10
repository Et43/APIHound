// Copyright 2024 Specter Ops, Inc.
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

package model

// IngestSource describes the high-level category of data being ingested.
// It is carried via the X-Ingest-Source HTTP header on file-upload requests.
type IngestSource string

const (
	IngestSourceADOrAzure IngestSource = "default"
	IngestSourceAPI       IngestSource = "api"
)

// IsValid returns true when the value is one of the accepted ingest sources.
func (s IngestSource) IsValid() bool {
	switch s {
	case IngestSourceADOrAzure, IngestSourceAPI:
		return true
	default:
		return false
	}
}
