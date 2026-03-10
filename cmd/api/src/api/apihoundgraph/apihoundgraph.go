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

package apihoundgraph

import "github.com/specterops/dawgs/graph"

//TODO: Move styling responsibilities to the UI or move shared styling definitions to a cue file to generate from one source of truth

type APIHoundGraphGlyph struct {
	Angle    int                        `json:"angle,omitempty"`
	Blink    bool                       `json:"blink,omitempty"`
	Border   *APIHoundGraphItemBorder `json:"border,omitempty"`
	Color    string                     `json:"color,omitempty"`
	FontIcon *APIHoundGraphFontIcon   `json:"fontIcon,omitempty"`
	Image    string                     `json:"image,omitempty"`
	Label    *APIHoundGraphLabel      `json:"label,omitempty"`
	Position string                     `json:"position,omitempty"` //Needs to be a compass direction (ne by default)
	Radius   int                        `json:"radius,omitempty"`
	Size     int                        `json:"size,omitempty"` //Defaults to 1
}

type APIHoundGraphItemBorder struct {
	Color string `json:"color,omitempty"`
}

type APIHoundGraphFontIcon struct {
	Color      string `json:"color,omitempty"`
	FontFamily string `json:"fontFamily,omitempty"`
	Text       string `json:"text,omitempty"`
}

type APIHoundGraphLabel struct {
	Bold       bool   `json:"bold,omitempty"`
	Color      string `json:"color,omitempty"`
	FontFamily string `json:"fontFamily,omitempty"`
	Text       string `json:"text,omitempty"`
}

type APIHoundGraphItem struct {
	Color  string                  `json:"color,omitempty"`
	Data   map[string]any          `json:"data,omitempty"`
	Fade   bool                    `json:"fade,omitempty"`
	Glyphs *[]APIHoundGraphGlyph `json:"glyphs,omitempty"`
}

type APIHoundGraphNodeBorder struct {
	Color     string `json:"color,omitempty"`
	LineStyle string `json:"lineStyle,omitempty"` //solid or dashed
	Width     int    `json:"width,omitempty"`
}

type APIHoundGraphNodeCoords struct {
	Latitude  int `json:"lat,omitempty"`
	Longitude int `json:"lng,omitempty"`
}

type APIHoundGraphNodeHalo struct {
	Color  string `json:"color,omitempty"`
	Radius int    `json:"radius,omitempty"`
	Width  int    `json:"width,omitempty"`
}

type APIHoundGraphNodeLabel struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Bold            bool   `json:"bold,omitempty"`
	Center          bool   `json:"center,omitempty"`
	Color           string `json:"color,omitempty"`
	FontFamily      string `json:"fontFamily,omitempty"`
	FontSize        int    `json:"fontSize,omitempty"`
	Text            string `json:"text,omitempty"`
}

type APIHoundGraphNode struct {
	*APIHoundGraphItem
	Border      *APIHoundGraphNodeBorder `json:"border,omitempty"`
	Coordinates *APIHoundGraphNodeCoords `json:"coordinates,omitempty"`
	Cutout      bool                       `json:"cutout,omitempty"`
	FontIcon    *APIHoundGraphFontIcon   `json:"fontIcon,omitempty"`
	Halos       *[]APIHoundGraphNodeHalo `json:"halos,omitempty"`
	Image       string                     `json:"image,omitempty"`
	Label       *APIHoundGraphNodeLabel  `json:"label,omitempty"`
	Shape       string                     `json:"shape,omitempty"`
	Size        int                        `json:"size,omitempty"`
}

type APIHoundGraphLinkLabel struct {
	BackgroundColor string `json:"backgroundColor,omitempty"`
	Bold            bool   `json:"bold,omitempty"`
	Color           string `json:"color,omitempty"`
	FontFamily      string `json:"fontFamily,omitempty"`
	FontSize        int    `json:"fontSize,omitempty"`
	Text            string `json:"text,omitempty"`
}

type APIHoundGraphLinkEnd struct {
	Arrow   bool                      `json:"arrow,omitempty"`
	BackOff int                       `json:"backOff,omitempty"`
	Color   string                    `json:"color,omitempty"`
	Glyphs  *[]APIHoundGraphGlyph   `json:"glyphs,omitempty"`
	Label   *APIHoundGraphLinkLabel `json:"label,omitempty"`
}

type APIHoundGraphLinkFlow struct {
	Velocity int `json:"velocity,omitempty"`
}

type APIHoundGraphLink struct {
	*APIHoundGraphItem
	End1      *APIHoundGraphLinkEnd   `json:"end1,omitempty"`
	End2      *APIHoundGraphLinkEnd   `json:"end2,omitempty"`
	Flow      *APIHoundGraphLinkFlow  `json:"flow,omitempty"`
	ID1       string                    `json:"id1,omitempty"`
	ID2       string                    `json:"id2,omitempty"`
	Label     *APIHoundGraphLinkLabel `json:"label,omitempty"`
	LineStyle string                    `json:"lineStyle,omitempty"`
	Width     int                       `json:"width,omitempty"`
}

func (s *APIHoundGraphNode) SetNodeStyle(nType string) {
	s.SetIcon(nType)
	s.SetBackground(nType)
}

func (s *APIHoundGraphNode) SetIcon(nType string) {
	switch nType {
	case "AZApp":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-window-restore",
		}
	case "AZVMScaleSet":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-server",
		}
	case "AZDevice":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-desktop",
		}
	case "AZFunctionApp":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-bolt",
		}
	case "AZGroup":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-users",
		}
	case "AZKeyVault":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-lock",
		}
	case "AZManagementGroup":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-sitemap",
		}
	case "AZResourceGroup":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-cube",
		}
	case "AZRole":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-clipboard-list",
		}
	case "AZServicePrincipal":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-robot",
		}
	case "AZSubscription":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-key",
		}
	case "AZTenant":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-cloud",
		}
	case "AZUser":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-user",
		}
	case "AZVM":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-desktop",
		}
	case "AZManagedCluster":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-cubes",
		}
	case "AZContainerRegistry":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-box-open",
		}
	case "AZWebApp":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-object-group",
		}
	case "AZLogicApp":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-sitemap",
		}
	case "AZAutomationAccount":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-cog",
		}
	case "User":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-user",
		}
	case "Group":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-users",
		}
	case "Computer":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-desktop",
		}
	case "Container":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-box",
		}
	case "Domain":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-globe",
		}
	case "OU":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-sitemap",
		}
	case "GPO":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-list",
		}
	case "AIACA":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-arrows-left-right-to-line",
		}
	case "RootCA":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-landmark",
		}
	case "EnterpriseCA":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-building",
		}
	case "NTAuthStore":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-store",
		}
	case "CertTemplate":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-id-card",
		}
	case "IssuancePolicy":
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-clipboard-check",
		}
	case "Meta":
		if tier, ok := s.Data["admintier"]; ok {
			if tier.(int64) == 0 {
				s.Image = "/ui/metat0.png"
			} else {
				s.Image = "/ui/meta.png"
			}
		} else {
			s.Image = "/ui/meta.png"
		}
	default:
		s.FontIcon = &APIHoundGraphFontIcon{
			Text: "fas fa-question",
		}
	}
}

func (s *APIHoundGraphNode) SetBackground(nType string) {
	switch nType {
	case "AZApp":
		s.Color = "#03FC84"
	case "AZVMScaleSet":
		s.Color = "#007CD0"
	case "AZDevice":
		s.Color = "#B18FCF"
	case "AZFunctionApp":
		s.Color = "#F4BA44"
	case "AZGroup":
		s.Color = "#F57C9B"
	case "AZKeyVault":
		s.Color = "#ED658C"
	case "AZManagementGroup":
		s.Color = "#BD93D8"
	case "AZResourceGroup":
		s.Color = "#89BD9E"
	case "AZRole":
		s.Color = "#ED8537"
	case "AZServicePrincipal":
		s.Color = "#C1D6D6"
	case "AZSubscription":
		s.Color = "#D2CCA1"
	case "AZTenant":
		s.Color = "#54F2F2"
	case "AZUser":
		s.Color = "#34D2EB"
	case "AZVM":
		s.Color = "#F9ADA0"
	case "AZManagedCluster":
		s.Color = "#326CE5"
	case "AZContainerRegistry":
		s.Color = "#0885D7"
	case "AZWebApp":
		s.Color = "#4696E9"
	case "AZLogicApp":
		s.Color = "#9EE047"
	case "AZAutomationAccount":
		s.Color = "#F4BA44"
	case "User":
		s.Color = "#17E625"
	case "Group":
		s.Color = "#DBE617"
	case "Computer":
		s.Color = "#E67873"
	case "Container":
		s.Color = "#F79A78"
	case "Domain":
		s.Color = "#17E6B9"
	case "OU":
		s.Color = "#FFAA00"
	case "GPO":
		s.Color = "#998EFD"
	case "AIACA":
		s.Color = "#9769F0"
	case "RootCA":
		s.Color = "#6968E8"
	case "EnterpriseCA":
		s.Color = "#4696E9"
	case "NTAuthStore":
		s.Color = "#D575F5"
	case "CertTemplate":
		s.Color = "#B153F3"
	case "Meta":
		s.Color = "#000"
	default:
		s.Color = "#EEE"
	}
}

func (s *APIHoundGraphNode) SetNodeType(kind graph.Kind) {
	if s.Data == nil {
		s.Data = make(map[string]any)
	}
	s.Data["nodetype"] = kind
}
