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

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

// Client is a lightweight HTTP client for the BloodHound file-upload API.
type Client struct {
	serverURL  string
	token      string
	httpClient *http.Client
}

// IngestJob represents the response from the start-job endpoint.
type IngestJob struct {
	ID int64 `json:"id"`
}

// UploadOptions holds per-file upload metadata.
type UploadOptions struct {
	FileName        string
	OpCoName        string
	OpCoDescription string
}

// NewClient creates a new BloodHound API client.
func NewClient(serverURL string, token string) *Client {
	return &Client{
		serverURL: strings.TrimRight(serverURL, "/"),
		token:     token,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

// StartIngestJob creates a new file-upload job and returns its ID.
func (s *Client) StartIngestJob() (int64, error) {
	requestURL := s.serverURL + "/api/v2/file-upload/start"

	request, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		return 0, fmt.Errorf("creating start-job request: %w", err)
	}
	s.setAuthHeader(request)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return 0, fmt.Errorf("sending start-job request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		return 0, fmt.Errorf("start-job returned %d: %s", response.StatusCode, string(body))
	}

	var wrapper struct {
		Data IngestJob `json:"data"`
	}
	if err := json.NewDecoder(response.Body).Decode(&wrapper); err != nil {
		return 0, fmt.Errorf("decoding start-job response: %w", err)
	}

	return wrapper.Data.ID, nil
}

// UploadFile uploads a single API specification file to an existing ingest job.
func (s *Client) UploadFile(jobID int64, fileContent []byte, opts UploadOptions) error {
	requestURL := fmt.Sprintf("%s/api/v2/file-upload/%d", s.serverURL, jobID)

	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(fileContent))
	if err != nil {
		return fmt.Errorf("creating upload request: %w", err)
	}
	s.setAuthHeader(request)

	// Determine content type from file extension
	contentType := contentTypeForFile(opts.FileName)
	request.Header.Set("Content-Type", contentType)
	request.Header.Set("X-Ingest-Source", "api")
	request.Header.Set("X-File-Upload-Name", opts.FileName)

	if opts.OpCoName != "" {
		request.Header.Set("X-OpCo-Name", opts.OpCoName)
	}
	if opts.OpCoDescription != "" {
		request.Header.Set("X-OpCo-Description", opts.OpCoDescription)
	}

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending upload request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("upload returned %d: %s", response.StatusCode, string(body))
	}

	return nil
}

// EndIngestJob signals the server that all files have been uploaded for this job.
func (s *Client) EndIngestJob(jobID int64) error {
	requestURL := fmt.Sprintf("%s/api/v2/file-upload/%d/end", s.serverURL, jobID)

	request, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		return fmt.Errorf("creating end-job request: %w", err)
	}
	s.setAuthHeader(request)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("sending end-job request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(response.Body)
		return fmt.Errorf("end-job returned %d: %s", response.StatusCode, string(body))
	}

	return nil
}

func (s *Client) setAuthHeader(request *http.Request) {
	if s.token != "" {
		request.Header.Set("Authorization", "Bearer "+s.token)
	}
}

// contentTypeForFile returns the MIME type appropriate for the given filename.
func contentTypeForFile(fileName string) string {
	extension := strings.ToLower(filepath.Ext(fileName))
	switch extension {
	case ".yaml", ".yml":
		return "application/x-yaml"
	default:
		return "application/json"
	}
}
