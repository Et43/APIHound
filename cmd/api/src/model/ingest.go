// Copyright 2023 Specter Ops, Inc.
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

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/specterops/bloodhound/cmd/api/src/database/types/null"
)

// IngestTaskMetadata holds optional enrichment metadata that travels with an
// ingest task from upload-time through to graph conversion. Stored as a JSONB
// column so new fields can be added without schema migrations.
type IngestTaskMetadata struct {
	// OpCoName is the operational/business unit that owns this API collection.
	OpCoName string `json:"opco_name,omitempty"`

	// OpCoDescription is an optional human-readable description of the OpCo.
	OpCoDescription string `json:"opco_description,omitempty"`
}

// Scan implements the sql.Scanner interface for reading JSONB from the database.
func (s *IngestTaskMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	default:
		return fmt.Errorf("IngestTaskMetadata.Scan: unsupported type %T", value)
	}

	return json.Unmarshal(bytes, s)
}

// Value implements the driver.Valuer interface for writing JSONB to the database.
func (s IngestTaskMetadata) Value() (driver.Value, error) {
	data, err := json.Marshal(s)
	if err != nil {
		return nil, err
	}
	return string(data), nil
}

type IngestTask struct {
	StoredFileName   string              `json:"file_name"`
	OriginalFileName string              `json:"original_file_name"`
	RequestGUID      string              `json:"request_guid"`
	JobId            null.Int64          `json:"task_id" gorm:"column:task_id"`
	FileType         FileType            `json:"file_type"`
	Metadata         *IngestTaskMetadata `json:"metadata,omitempty" gorm:"type:jsonb;default:null"`

	BigSerial
}

type IngestTasks []IngestTask

type FileType int

const (
	FileTypeJson FileType = iota
	FileTypeZip
	FileTypeAPIJson
	FileTypeAPIYaml
)

// IsAPISource returns true when the file type represents API environment data
// rather than AD/Azure graph data.
func (s FileType) IsAPISource() bool {
	return s == FileTypeAPIJson || s == FileTypeAPIYaml
}
