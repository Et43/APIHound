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

package fixtures

import (
	"context"
	"fmt"
	"log"

	"github.com/specterops/apihound/cmd/api/src/config"

	"github.com/specterops/apihound/cmd/api/src/auth"

	"github.com/specterops/apihound/cmd/api/src/bootstrap"
	"github.com/specterops/apihound/cmd/api/src/database"
	"github.com/specterops/apihound/cmd/api/src/test/integration"
	"github.com/specterops/apihound/packages/go/lab"
)

var PostgresFixture = lab.NewFixture(func(harness *lab.Harness) (*database.ApihoundDB, error) {
	testCtx := context.Background()
	if labConfig, ok := lab.Unpack(harness, ConfigFixture); !ok {
		return nil, fmt.Errorf("unable to unpack ConfigFixture")
	} else if pgdb, err := database.OpenDatabase(labConfig.Database.PostgreSQLConnectionString()); err != nil {
		return nil, err
	} else if err := integration.Prepare(testCtx, database.NewApihoundDB(pgdb, auth.NewIdentityResolver())); err != nil {
		return nil, fmt.Errorf("failed ensuring database: %v", err)
	} else if err := bootstrap.MigrateDB(testCtx, labConfig, database.NewApihoundDB(pgdb, auth.NewIdentityResolver()), config.NewDefaultAdminConfiguration); err != nil {
		return nil, fmt.Errorf("failed migrating database: %v", err)
	} else if err := bootstrap.PopulateExtensionData(testCtx, database.NewApihoundDB(pgdb, auth.NewIdentityResolver())); err != nil {
		return nil, fmt.Errorf("failed populating extension data: %v", err)
	} else {
		return database.NewApihoundDB(pgdb, auth.NewIdentityResolver()), nil
	}
}, nil)

func init() {
	if err := lab.SetDependency(PostgresFixture, ConfigFixture); err != nil {
		log.Fatalln(err)
	}
}
