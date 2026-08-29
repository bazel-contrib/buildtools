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

// Warnings for using deprecated functions

package warn

import (
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
)

// acceptsNameArgument checks whether a function can accept a named argument called "name",
// either directly or via **kwargs.
func acceptsNameArgument(def *build.DefStmt) bool {
	for _, param := range def.Params {
		if name, op := build.GetParamName(param); name == "name" || op == "**" {
			return true
		}
	}
	return false
}

func unnamedMacroWarning(f *build.File, fileReader *FileReader) []*LinterFinding {
	if f.Type != build.TypeBzl {
		return nil
	}

	macroAnalyzer := NewMacroAnalyzer(fileReader)

	findings := []*LinterFinding{}
	for _, stmt := range f.Stmt {
		def, ok := stmt.(*build.DefStmt)
		if !ok {
			continue
		}

		if strings.HasPrefix(def.Name, "_") || acceptsNameArgument(def) {
			continue
		}

		report, err := macroAnalyzer.AnalyzeFn(f, def)
		if err != nil {
			// TODO: Analysis errors are simply ignored as buildifier does not currently handle errors.
			continue
		}
		if !report.CanProduceTargets() {
			continue
		}
		msg := fmt.Sprintf(`The macro %q should have a keyword argument called "name".
It is considered a macro as it may produce targets via calls:
%s

By convention, every public macro needs a "name" argument (even if it doesn't use it).
This is important for tooling and automation.

* If this function is a helper function that's not supposed to be used outside of this file,
  please make it private (e.g. rename it to "_%s").
* Otherwise, add a "name" argument. If possible, use that name when calling other macros/rules.`,
			def.Name,
			report.PrintableCallStack(),
			def.Name)
		finding := makeLinterFinding(def, msg)
		finding.End = def.ColonPos
		findings = append(findings, finding)
	}

	return findings
}
