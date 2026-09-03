/*
Copyright 2021 Google LLC

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

// Documentation generator
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/bazelbuild/buildtools/warn"
	"github.com/golang/protobuf/proto"

	docspb "github.com/bazelbuild/buildtools/warn/docs/proto"
)

func readWarningsFromFile(path string) (*docspb.Warnings, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	warnings := &docspb.Warnings{}
	if err := proto.UnmarshalText(string(content), warnings); err != nil {
		return nil, err
	}
	return warnings, nil
}

func isExistingWarning(name string) bool {
	for _, n := range warn.AllWarnings {
		if n == name {
			return true
		}
	}
	return false
}

func isDisabledWarning(name string) bool {
	if !isExistingWarning(name) {
		return false
	}
	for _, n := range warn.DefaultWarnings {
		if n == name {
			return false
		}
	}
	return true
}

func generateWarningsDocs(warnings *docspb.Warnings) string {
	var b bytes.Buffer

	b.WriteString(`# Buildifier warnings

Warning categories supported by buildifier's linter:

`)

	// Table of contents
	var names []string
	for _, w := range warnings.Warnings {
		names = append(names, w.Name...)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Fprintf(&b, "  * [`%s`](#%s)\n", n, n)
	}

	// Misc
	b.WriteString(`
### <a name="suppress"></a>How to disable warnings

All warnings can be disabled / suppressed / ignored by adding a special comment ` + "`" + `# buildifier: disable=<category_name>` + "`" + ` to
the expression that causes the warning. Historically comments with ` + "`" + `buildozer` + "`" + ` instead of
` + "`" + `buildifier` + "`" + ` are also supported, they are equivalent.

To disable multiple warnings, you can specify them as a comma-separated list on a single line:

` + "```" + `python
# buildifier: disable=warning1,warning2
` + "```" + `

Or place them on separate lines:

` + "```" + `python
# buildifier: disable=warning1
# buildifier: disable=warning2
` + "```" + `

#### Examples

` + "```" + `python
# buildifier: disable=no-effect
"""
A multiline comment as a string literal.

Docstrings don't trigger the warning if they are first statements of a file or a function.
"""

if debug:
    print("Debug information:", foo)  # buildifier: disable=print
` + "```\n")

	// Individual warnings
	sort.Slice(warnings.Warnings, func(i, j int) bool {
		return strings.Compare(warnings.Warnings[i].Name[0], warnings.Warnings[j].Name[0]) < 0
	})
	for _, w := range warnings.Warnings {
		// Header
		b.WriteString("\n--------------------------------------------------------------------------------\n\n## ")
		for _, n := range w.Name {
			fmt.Fprintf(&b, "<a name=%q></a>", n)
		}
		fmt.Fprintf(&b, "%s\n\n", w.Header)

		// Name(s)
		if len(w.Name) == 1 {
			fmt.Fprintf(&b, "  * Category name: `%s`\n", w.Name[0])
		} else {
			b.WriteString("  * Category names:\n")
			for _, n := range w.Name {
				fmt.Fprintf(&b, "    * `%s`\n", n)
			}
		}

		// Bazel --incompatible flag
		if w.BazelFlag != "" {
			label := fmt.Sprintf("`%s`", w.BazelFlag)
			if w.BazelFlagLink != "" {
				label = fmt.Sprintf("[%s](%s)", label, w.BazelFlagLink)
			}
			fmt.Fprintf(&b, "  * Flag in Bazel: %s\n", label)
		}

		// Automatic fix
		fix := "no"
		if w.Autofix {
			fix = "yes"
		}
		fmt.Fprintf(&b, "  * Automatic fix: %s\n", fix)

		// Disabled by default
		if isDisabledWarning(w.Name[0]) {
			b.WriteString("  * [Disabled by default](buildifier/README.md#linter)\n")
		}

		// Non-existent
		if !isExistingWarning(w.Name[0]) {
			b.WriteString("  * Not supported by the latest version of Buildifier\n")
		}

		// Suppress the warning
		b.WriteString("  * [Suppress the warning](#suppress): ")
		for i, n := range w.Name {
			if i != 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "`# buildifier: disable=%s`", n)
		}
		b.WriteString("\n")

		// Description
		fmt.Fprintf(&b, "\n%s\n", w.Description)
	}
	return b.String()
}

func writeWarningsDocs(docs, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := f.WriteString(docs); err != nil {
		return err
	}
	return f.Close()
}

func main() {
	flag.Parse()
	warnings, err := readWarningsFromFile(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	docs := generateWarningsDocs(warnings)
	if err := writeWarningsDocs(docs, flag.Arg(1)); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
