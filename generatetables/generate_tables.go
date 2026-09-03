/*
Copyright 2016 Google LLC

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

// generateTables is a tool that generates a go file from the Build language proto file.
// It generates a Go map to find the type of an attribute.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"

	buildpb "github.com/bazel-contrib/buildtools/v10/build_proto"
	"github.com/golang/protobuf/proto"
)

var inputPath = flag.String("input", "", "input file")
var outputPath = flag.String("output", "", "output file")

// bazelBuildLanguage reads a proto file and returns a BuildLanguage object.
func bazelBuildLanguage(file string) (*buildpb.BuildLanguage, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Cannot read %s: %s\n", file, err)
		return nil, err
	}

	lang := &buildpb.BuildLanguage{}
	if err := proto.Unmarshal(data, lang); err != nil {
		return nil, err
	}
	return lang, nil
}

// generateTable returns a map that associate a type for each attribute name found in Bazel.
func generateTable(rules []*buildpb.RuleDefinition) map[string]buildpb.Attribute_Discriminator {
	types := make(map[string]buildpb.Attribute_Discriminator)
	for _, r := range rules {
		for _, attr := range r.Attribute {
			types[*attr.Name] = *attr.Type
		}
	}

	// Because of inconsistencies in bazel, we need a few exceptions.
	types["resources"] = buildpb.Attribute_LABEL_LIST
	types["out"] = buildpb.Attribute_STRING
	types["outs"] = buildpb.Attribute_STRING_LIST
	types["stamp"] = buildpb.Attribute_TRISTATE
	types["strip"] = buildpb.Attribute_BOOLEAN

	// Surprisingly, the name argument is missing.
	types["name"] = buildpb.Attribute_STRING

	// package arguments are also not listed in the proto file
	types["default_hdrs_check"] = buildpb.Attribute_STRING
	types["default_visibility"] = types["visibility"]
	types["default_copts"] = buildpb.Attribute_STRING_LIST
	types["default_deprecation"] = types["deprecation"]
	types["default_testonly"] = types["testonly"]
	types["default_applicable_licenses"] = buildpb.Attribute_LABEL_LIST
	types["default_package_metadata"] = buildpb.Attribute_LABEL_LIST
	types["features"] = buildpb.Attribute_STRING_LIST

	types["extra_srcs"] = types["srcs"]
	types["pytype_deps"] = types["deps"]

	// Make sure we always have this.
	_, ok := types["package_metadata"]
	if !ok {
		types["package_metadata"] = buildpb.Attribute_LABEL_LIST
	}

	// The following attributes used to exist in the native Python rules but were removed
	// during Starlarkification. Add them here for backwards compatibility, at least for now.
	types["api_version"] = buildpb.Attribute_INTEGER
	types["bootstrap_template"] = buildpb.Attribute_LABEL
	types["buildpar"] = buildpb.Attribute_BOOLEAN
	types["coverage_tool"] = buildpb.Attribute_LABEL
	types["files"] = buildpb.Attribute_LABEL_LIST
	types["has_services"] = buildpb.Attribute_BOOLEAN
	types["interpreter"] = buildpb.Attribute_LABEL
	types["interpreter_path"] = buildpb.Attribute_STRING
	types["launcher"] = buildpb.Attribute_LABEL
	types["launcher_uses_whole_archive"] = buildpb.Attribute_BOOLEAN
	types["main"] = buildpb.Attribute_LABEL
	types["paropts"] = buildpb.Attribute_STRING_LIST
	types["python_version"] = buildpb.Attribute_STRING
	types["srcs_version"] = buildpb.Attribute_STRING
	types["stub_shebang"] = buildpb.Attribute_STRING
	
	// The following attributes used to exist in the native CC rules but were removed
	// during Starlarkification. Add them here for backwards compatibility, at least for now.
	types["copts"] = buildpb.Attribute_STRING_LIST
	types["linkstamp"] = buildpb.Attribute_LABEL

	return types
}

func main() {
	flag.Parse()
	if *inputPath == "" {
		log.Fatal("No input file specified")
	}
	lang, err := bazelBuildLanguage(*inputPath)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	types := generateTable(lang.Rule)

	// sort the keys to get deterministic output
	keys := make([]string, 0, len(types))
	for i := range types {
		keys = append(keys, i)
	}
	sort.Strings(keys)

	f, err := os.Create(*outputPath)
	if err != nil {
		log.Fatalf("%s\n", err)
	}
	defer f.Close()

	// print
	fmt.Fprintf(f, `// Generated file, do not edit.
package lang

import buildpb "github.com/bazel-contrib/buildtools/v10/build_proto"

var TypeOf = map[string]buildpb.Attribute_Discriminator{
`)
	for _, attr := range keys {
		fmt.Fprintf(f, "	\"%s\":	buildpb.Attribute_%s,\n", attr, types[attr])
	}
	fmt.Fprintf(f, "}\n")
}
