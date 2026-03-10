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

package graphschema

import (
"github.com/specterops/apihound/packages/go/graphschema/common"
"github.com/specterops/dawgs/graph"
)

const (
DefaultMissingName     = "NO NAME"
DefaultMissingObjectId = "NO OBJECT ID"
)

func CombinedGraphSchema(name string) graph.Graph {
return graph.Graph{
Name:  name,
Nodes: common.NodeKinds(),
Edges: common.Relationships(),
NodeConstraints: []graph.Constraint{{
Field: common.ObjectID.String(),
Type:  graph.BTreeIndex,
}},
NodeIndexes: []graph.Index{
{
Field: common.Name.String(),
Type:  graph.TextSearchIndex,
},
{
Field: common.SystemTags.String(),
Type:  graph.TextSearchIndex,
},
{
Field: common.UserTags.String(),
Type:  graph.TextSearchIndex,
},
},
}
}

func DefaultGraph() graph.Graph {
return CombinedGraphSchema("default")
}

func DefaultGraphSchema() graph.Schema {
defaultGraph := DefaultGraph()

return graph.Schema{
Graphs: []graph.Graph{
defaultGraph,
},
DefaultGraph: defaultGraph,
}
}
