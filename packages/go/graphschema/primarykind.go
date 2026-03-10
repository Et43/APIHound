// Copyright 2025 Specter Ops, Inc.
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

package graphschema

import (
"github.com/specterops/apihound/packages/go/graphschema/common"
"github.com/specterops/dawgs/graph"
)

var (
meta         = graph.StringKind("Meta")
metaDetail   = graph.StringKind("MetaDetail")
metaIncludes = graph.StringKind("MetaIncludes")
metaKinds    = []graph.Kind{meta, metaDetail, metaIncludes}

unknownKind = graph.StringKind("Unknown")

ValidKinds = buildValidKinds()
)

func buildValidKinds() map[graph.Kind]bool {
var (
validKinds = make(map[graph.Kind]bool)
kindSlices = []graph.Kinds{
common.NodeKinds(),
common.Relationships(),
}
)

for _, kindSlice := range kindSlices {
for _, kind := range kindSlice {
validKinds[kind] = true
}
}

return validKinds
}

type ValidPrimaryKinds map[graph.Kind]bool

func PrimaryNodeKind(validPrimaryKinds ValidPrimaryKinds, kinds graph.Kinds) graph.Kind {
resultKind := unknownKind

if validPrimaryKinds == nil {
validPrimaryKinds = ValidKinds
}

for _, kind := range kinds {
if kind.Is(metaKinds...) {
return meta
} else if validPrimaryKinds[kind] {
resultKind = kind
}
}

return resultKind
}
