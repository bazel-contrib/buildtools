package warn

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/tables"
)

func symbolLoadLocationWarning(f *build.File) []*LinterFinding {
	var findings []*LinterFinding

	for stmtIndex := 0; stmtIndex < len(f.Stmt); stmtIndex++ {
		load, ok := f.Stmt[stmtIndex].(*build.LoadStmt)
		if !ok {
			continue
		}

		for i := 0; i < len(load.From); i++ {
			from := load.From[i]

			expected, ok := tables.AllowedSymbolLoadLocations[from.Name]
			if !ok || expected[load.Module.Value] {
				continue
			}

			var f *LinterFinding
			if len(expected) == 1 {
				var loc string
				for l := range expected {
					loc = l
					break
				}
				f = makeLinterFinding(from, fmt.Sprintf("Symbol %q must be loaded from %s.", from.Name, loc))
			} else {
				locs := make([]string, 0, len(expected))
				for l := range expected {
					locs = append(locs, l)
				}
				slices.Sort(locs)
				f = makeLinterFinding(from, fmt.Sprintf("Symbol %q must be loaded from one of the allowed locations: %s.", from.Name, strings.Join(locs, ", ")))
			}
			findings = append(findings, f)
		}

	}
	return findings
}
