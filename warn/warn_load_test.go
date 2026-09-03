/*
Copyright 2020 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    https://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package warn

import (
	"testing"

	"github.com/bazel-contrib/buildtools/v10/tables"
)

func TestWarnLoadLocation(t *testing.T) {
	tables.AllowedSymbolLoadLocations["s1"] = map[string]bool{":z.bzl": true}
	tables.AllowedSymbolLoadLocations["s3"] = map[string]bool{":x.bzl": true, ":y.bzl": true}
	tables.AllowedSymbolLoadLocations["s4"] = map[string]bool{":a.bzl": true}
	checkFindingsAndFix(t, "allowed-symbol-load-locations", `
load(":f.bzl", "s1", "s2")
load(":a.bzl", "s3")
load(":a.bzl", "s4")
`, `
load(":f.bzl", "s1", "s2")
load(":a.bzl", "s3")
load(":a.bzl", "s4")
`,
		[]string{
			":1: Symbol \"s1\" must be loaded from :z.bzl.",
			":2: Symbol \"s3\" must be loaded from one of the allowed locations: :x.bzl, :y.bzl.",
		},
		scopeEverywhere)
}
