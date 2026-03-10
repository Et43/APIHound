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

package database

import (
	"context"

	"github.com/specterops/apihound/cmd/api/src/model"
	"github.com/specterops/apihound/cmd/api/src/model/appcfg"
	"gorm.io/gorm"
)

func (s *ApihoundDB) GetFlag(ctx context.Context, id int32) (appcfg.FeatureFlag, error) {
	var flag appcfg.FeatureFlag
	return flag, CheckError(s.db.WithContext(ctx).Find(&flag, id))
}

func (s *ApihoundDB) GetFlagByKey(ctx context.Context, key string) (appcfg.FeatureFlag, error) {
	var flag appcfg.FeatureFlag
	return flag, CheckError(s.db.WithContext(ctx).Where("key = ?", key).First(&flag))
}

func (s *ApihoundDB) GetAllFlags(ctx context.Context) ([]appcfg.FeatureFlag, error) {
	var flags []appcfg.FeatureFlag
	return flags, CheckError(s.db.WithContext(ctx).Find(&flags))
}

func (s *ApihoundDB) SetFlag(ctx context.Context, flag appcfg.FeatureFlag) error {
	var (
		auditEntry = model.AuditEntry{
			Action: model.AuditLogActionToggleEarlyAccessFeatureFlag,
			Model:  flag,
		}
	)

	return s.MaybeAuditableTransaction(ctx, !flag.UserUpdatable, auditEntry, func(tx *gorm.DB) error {
		return CheckError(tx.Save(&flag))
	})
}
