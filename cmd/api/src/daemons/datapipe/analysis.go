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

package datapipe

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/specterops/apihound/cmd/api/src/config"
	"github.com/specterops/apihound/cmd/api/src/database"
	"github.com/specterops/apihound/cmd/api/src/model/appcfg"
	"github.com/specterops/apihound/cmd/api/src/services/agi"
	"github.com/specterops/apihound/cmd/api/src/services/dataquality"
	"github.com/specterops/dawgs/graph"
)

var (
	ErrAnalysisFailed             = errors.New("analysis failed")
	ErrAnalysisPartiallyCompleted = errors.New("analysis partially completed")
)

func RunAnalysisOperations(ctx context.Context, db database.Database, graphDB graph.Database, _ config.Configuration) error {
	var (
		collectedErrors   []error
		tieringEnabled    = appcfg.GetTieringEnabled(ctx, db)
		agiFailed         = false
		dataQualityFailed = false
	)

	// TODO: Wire in API-specific analysis (Phase 5)

	if errs := TagAssetGroupsAndTierZero(ctx, db, graphDB); len(errs) > 0 {
		for _, err := range errs {
			collectedErrors = append(collectedErrors, fmt.Errorf("tagging asset groups failed: %%w", err))
		}
	}

	if !tieringEnabled {
		if err := agi.RunAssetGroupIsolationCollections(ctx, db, graphDB); err != nil {
			collectedErrors = append(collectedErrors, fmt.Errorf("asset group isolation collection failed: %%w", err))
			agiFailed = true
		}
	}

	if err := dataquality.SaveDataQuality(ctx, db, graphDB); err != nil {
		collectedErrors = append(collectedErrors, fmt.Errorf("error saving data quality stat: %%v", err))
		dataQualityFailed = true
	}

	if len(collectedErrors) > 0 {
		for _, err := range collectedErrors {
			slog.ErrorContext(ctx, fmt.Sprintf("Analysis error encountered: %%v", err))
		}
	}

	if agiFailed && dataQualityFailed {
		return ErrAnalysisFailed
	} else if agiFailed || dataQualityFailed {
		return ErrAnalysisPartiallyCompleted
	}

	return nil
}
