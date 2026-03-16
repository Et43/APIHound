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

package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/specterops/bloodhound/cmd/collector/client"
	"github.com/specterops/bloodhound/cmd/collector/manifest"
)

// Execute parses CLI flags and runs the collector.
func Execute() error {
	var (
		manifestPath string
		serverURL    string
		token        string
		dryRun       bool
		verbose      bool
	)

	collectCmd := flag.NewFlagSet("collect", flag.ExitOnError)
	collectCmd.StringVar(&manifestPath, "manifest", "", "Path to the manifest file (YAML or JSON)")
	collectCmd.StringVar(&serverURL, "server", "", "BloodHound server URL (e.g. https://bloodhound:8080)")
	collectCmd.StringVar(&token, "token", "", "BloodHound API bearer token")
	collectCmd.BoolVar(&dryRun, "dry-run", false, "Parse and validate without uploading")
	collectCmd.BoolVar(&verbose, "verbose", false, "Enable verbose output")

	if len(os.Args) < 2 {
		printUsage()
		return fmt.Errorf("expected 'collect' subcommand")
	}

	switch os.Args[1] {
	case "collect":
		if err := collectCmd.Parse(os.Args[2:]); err != nil {
			return err
		}
		return runCollect(manifestPath, serverURL, token, dryRun, verbose)
	case "version":
		fmt.Println("apihound-collector v0.1.0")
		return nil
	case "help", "--help", "-h":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand: %s", os.Args[1])
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `APIHound Collector — ingest OpenAPI/Swagger specs into BloodHound

Usage:
  apihound-collector collect [flags]
  apihound-collector version
  apihound-collector help

Flags for 'collect':
  --manifest <path>   Path to the manifest file (YAML or JSON) [required]
  --server  <url>     BloodHound server URL [required unless --dry-run]
  --token   <token>   BloodHound API bearer token [required unless --dry-run]
  --dry-run           Parse and validate the manifest without uploading
  --verbose           Enable verbose output
`)
}

func runCollect(manifestPath string, serverURL string, token string, dryRun bool, verbose bool) error {
	if manifestPath == "" {
		return fmt.Errorf("--manifest is required")
	}

	if !dryRun {
		if serverURL == "" {
			return fmt.Errorf("--server is required (use --dry-run to skip upload)")
		}
		if token == "" {
			return fmt.Errorf("--token is required (use --dry-run to skip upload)")
		}
	}

	// Load and validate the manifest
	manifestDir := filepath.Dir(manifestPath)

	collectorManifest, err := manifest.LoadFromFile(manifestPath)
	if err != nil {
		return fmt.Errorf("loading manifest: %w", err)
	}

	if err := collectorManifest.Validate(manifestDir); err != nil {
		return fmt.Errorf("validating manifest: %w", err)
	}

	// Count total specs for reporting
	var totalSpecs int
	for _, opCo := range collectorManifest.OpCos {
		totalSpecs += len(opCo.Specs)
	}

	fmt.Printf("Manifest loaded: %d OpCo(s), %d spec(s)\n", len(collectorManifest.OpCos), totalSpecs)

	if dryRun {
		fmt.Println("\n[dry-run] Validating spec files...")
		return validateSpecs(collectorManifest, manifestDir, verbose)
	}

	// Upload specs to BloodHound
	bloodhoundClient := client.NewClient(serverURL, token)
	return uploadSpecs(bloodhoundClient, collectorManifest, manifestDir, verbose)
}

func validateSpecs(collectorManifest *manifest.Manifest, manifestDir string, verbose bool) error {
	var (
		successCount int
		failureCount int
	)

	for _, opCo := range collectorManifest.OpCos {
		fmt.Printf("\nOpCo: %s\n", opCo.Name)

		for _, spec := range opCo.Specs {
			specPath := manifest.ResolveSpecPath(manifestDir, spec.Path)

			// Verify the file can be read
			fileInfo, err := os.Stat(specPath)
			if err != nil {
				fmt.Printf("  ✗ %s — %v\n", spec.Path, err)
				failureCount++
				continue
			}

			if verbose {
				fmt.Printf("  ✓ %s (%d bytes)\n", spec.Path, fileInfo.Size())
			} else {
				fmt.Printf("  ✓ %s\n", spec.Path)
			}
			successCount++
		}
	}

	fmt.Printf("\nValidation complete: %d passed, %d failed\n", successCount, failureCount)

	if failureCount > 0 {
		return fmt.Errorf("%d spec(s) failed validation", failureCount)
	}
	return nil
}

func uploadSpecs(bloodhoundClient *client.Client, collectorManifest *manifest.Manifest, manifestDir string, verbose bool) error {
	var (
		totalUploaded int
		totalFailed   int
	)

	for _, opCo := range collectorManifest.OpCos {
		fmt.Printf("\nProcessing OpCo: %s\n", opCo.Name)

		// Create a single ingest job per OpCo
		jobID, err := bloodhoundClient.StartIngestJob()
		if err != nil {
			fmt.Printf("  ✗ Failed to create ingest job: %v\n", err)
			totalFailed += len(opCo.Specs)
			continue
		}

		if verbose {
			fmt.Printf("  Created ingest job %d\n", jobID)
		}

		for _, spec := range opCo.Specs {
			specPath := manifest.ResolveSpecPath(manifestDir, spec.Path)

			fileContent, err := os.ReadFile(specPath)
			if err != nil {
				fmt.Printf("  ✗ %s — failed to read: %v\n", spec.Path, err)
				totalFailed++
				continue
			}

			fileName := filepath.Base(spec.Path)
			if spec.ServiceName != "" {
				// Preserve original extension but use the service name
				extension := filepath.Ext(fileName)
				fileName = spec.ServiceName + extension
			}

			uploadOpts := client.UploadOptions{
				FileName:        fileName,
				OpCoName:        opCo.Name,
				OpCoDescription: opCo.Description,
			}

			if err := bloodhoundClient.UploadFile(jobID, fileContent, uploadOpts); err != nil {
				fmt.Printf("  ✗ %s — upload failed: %v\n", spec.Path, err)
				totalFailed++
				continue
			}

			fmt.Printf("  ✓ %s\n", spec.Path)
			totalUploaded++
		}

		// End the ingest job
		if err := bloodhoundClient.EndIngestJob(jobID); err != nil {
			fmt.Printf("  ⚠ Failed to end ingest job %d: %v\n", jobID, err)
		} else if verbose {
			fmt.Printf("  Ingest job %d completed\n", jobID)
		}
	}

	fmt.Printf("\nUpload complete: %d uploaded, %d failed\n", totalUploaded, totalFailed)

	if totalFailed > 0 {
		return fmt.Errorf("%d spec(s) failed to upload", totalFailed)
	}
	return nil
}
