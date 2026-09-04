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

// Package warn implements functions that generate warnings for BUILD files.
package warn

import (
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/edit"
)

// LintMode is an enum representing a linter mode. Can be either "warn", "fix", or "suggest"
type LintMode int

const (
	// ModeWarn means only warnings should be returned for each finding.
	ModeWarn LintMode = iota
	// ModeFix means that all warnings that can be fixed automatically should be fixed and
	// no warnings should be returned for them.
	ModeFix
	// ModeSuggest means that automatic fixes shouldn't be applied, but instead corresponding
	// suggestions should be attached to all warnings that can be fixed automatically.
	ModeSuggest
)

// LinterFinding is a low-level warning reported by single linter/fixer functions.
type LinterFinding struct {
	Start       build.Position
	End         build.Position
	Message     string
	URL         string
	Replacement []LinterReplacement
}

// LinterReplacement is a low-level object returned by single fixer functions.
type LinterReplacement struct {
	Old *build.Expr
	New build.Expr
}

// A Finding is a warning reported by the analyzer. It may contain an optional suggested fix.
type Finding struct {
	File        *build.File
	Start       build.Position
	End         build.Position
	Category    string
	Message     string
	URL         string
	Actionable  bool
	AutoFixable bool
	Replacement *Replacement
}

// A Replacement is a suggested fix. Text between Start and End should be replaced with Content.
type Replacement struct {
	Description string
	Start       int
	End         int
	Content     string
}

func docURL(cat string) string {
	return "https://github.com/bazel-contrib/buildtools/blob/main/WARNINGS.md#" + cat
}

// makeFinding creates a Finding object
func makeFinding(f *build.File, start, end build.Position, cat, url, msg string, actionable bool, autoFixable bool, fix *Replacement) *Finding {
	if url == "" {
		url = docURL(cat)
	}
	return &Finding{
		File:        f,
		Start:       start,
		End:         end,
		Category:    cat,
		URL:         url,
		Message:     msg,
		Actionable:  actionable,
		AutoFixable: autoFixable,
		Replacement: fix,
	}
}

// makeLinterFinding creates a LinterFinding object
func makeLinterFinding(node build.Expr, message string, replacement ...LinterReplacement) *LinterFinding {
	start, end := node.Span()
	return &LinterFinding{
		Start:       start,
		End:         end,
		Message:     message,
		Replacement: replacement,
	}
}

// RuleWarningMap lists the warnings that run on a single rule.
// These warnings run only on BUILD files (not bzl files).
var RuleWarningMap = map[string]func(call *build.CallExpr, pkg string) *LinterFinding{}

// FileWarningMap lists the warnings that run on the whole file.
var FileWarningMap = map[string]func(f *build.File) []*LinterFinding{
	"allowed-symbol-load-locations": symbolLoadLocationWarning,
	"attr-applicable_licenses":      attrApplicableLicensesWarning,
	"attr-cfg":                      attrConfigurationWarning,
	"attr-license":                  attrLicenseWarning,
	"attr-licenses":                 attrLicensesWarning,
	"attr-non-empty":                attrNonEmptyWarning,
	"attr-output-default":           attrOutputDefaultWarning,
	"attr-single-file":              attrSingleFileWarning,
	"build-args-kwargs":             argsKwargsInBuildFilesWarning,
	"confusing-name":                confusingNameWarning,
	"constant-glob":                 constantGlobWarning,
	"ctx-actions":                   ctxActionsWarning,
	"ctx-args":                      contextArgsAPIWarning,
	"depset-items":                  depsetItemsWarning,
	"depset-iteration":              depsetIterationWarning,
	"depset-union":                  depsetUnionWarning,
	"dict-method-named-arg":         dictMethodNamedArgWarning,
	"dict-concatenation":            dictionaryConcatenationWarning,
	"duplicated-name":               duplicatedNameWarning,
	"filetype":                      fileTypeWarning,
	"function-docstring":            functionDocstringWarning,
	"function-docstring-header":     functionDocstringHeaderWarning,
	"function-docstring-args":       functionDocstringArgsWarning,
	"function-docstring-return":     functionDocstringReturnWarning,
	"integer-division":              integerDivisionWarning,
	"keyword-positional-params":     keywordPositionalParametersWarning,
	"list-append":                   listAppendWarning,
	"load":                          unusedLoadWarning,
	"make-location":                 makeLocationVariableWarning,
	"module-docstring":              moduleDocstringWarning,
	"name-conventions":              nameConventionsWarning,
	"native-build":                  nativeInBuildFilesWarning,
	"native-package":                nativePackageWarning,
	"no-effect":                     noEffectWarning,
	"output-group":                  outputGroupWarning,
	"overly-nested-depset":          overlyNestedDepsetWarning,
	"package-name":                  packageNameWarning,
	"package-on-top":                packageOnTopWarning,
	"print":                         printWarning,
	"provider-params":               providerParamsWarning,
	"redefined-variable":            redefinedVariableWarning,
	"repository-name":               repositoryNameWarning,
	"rule-impl-return":              ruleImplReturnWarning,
	"return-value":                  missingReturnValueWarning,
	"skylark-comment":               skylarkCommentWarning,
	"skylark-docstring":             skylarkDocstringWarning,
	"string-iteration":              stringIterationWarning,
	"uninitialized":                 uninitializedVariableWarning,
	"unreachable":                   unreachableStatementWarning,
	"unsorted-dict-items":           unsortedDictItemsWarning,
	"unused-variable":               unusedVariableWarning,
}

// MultiFileWarningMap lists the warnings that run on the whole file, but may use other files.
var MultiFileWarningMap = map[string]func(f *build.File, fileReader *FileReader) []*LinterFinding{
	"bzl-visibility":                     bzlVisibilityWarning,
	"deprecated-function":                deprecatedFunctionWarning,
	"git-repository":                     nativeGitRepositoryWarning,
	"http-archive":                       nativeHTTPArchiveWarning,
	"native-android":                     nativeAndroidRulesWarning,
	"native-cc-binary":                   NativeCcRulesWarning("cc_binary"),
	"native-cc-import":                   NativeCcRulesWarning("cc_import"),
	"native-cc-library":                  NativeCcRulesWarning("cc_library"),
	"native-cc-objc-import":              NativeCcRulesWarning("objc_import"),
	"native-cc-objc-library":             NativeCcRulesWarning("objc_library"),
	"native-cc-shared-library":           NativeCcRulesWarning("cc_shared_library"),
	"native-cc-test":                     NativeCcRulesWarning("cc_test"),
	"native-cc-toolchain":                NativeCcToolchainRulesWarning("cc_toolchain"),
	"native-cc-toolchain-suite":          NativeCcToolchainRulesWarning("cc_toolchain_suite"),
	"native-cc-fdo-prefetch-hints":       NativeCcToolchainRulesWarning("fdo_prefetch_hints"),
	"native-cc-fdo-profile":              NativeCcToolchainRulesWarning("fdo_profile"),
	"native-cc-memprof-profile":          NativeCcToolchainRulesWarning("memprof_profile"),
	"native-cc-propeller-optimize":       NativeCcToolchainRulesWarning("propeller_optimize"),
	"native-cc-common":                   NativeCcSymbolsWarning("cc_common", "cc_common"),
	"native-cc-debug-package-info":       NativeCcSymbolsWarning("DebugPackageInfo", "debug_package_info"),
	"native-cc-info":                     NativeCcSymbolsWarning("CcInfo", "cc_info"),
	"native-cc-shared-library-info":      NativeCcSymbolsWarning("CcSharedLibraryInfo", "cc_shared_library_info"),
	"native-cc-shared-library-hint-info": NativeCcSymbolsWarning("CcSharedLibraryHintInfo", "cc_shared_library_hint_info"),
	"native-java-binary":                 NativeJavaRulesWarning("java_binary"),
	"native-java-import":                 NativeJavaRulesWarning("java_import"),
	"native-java-library":                NativeJavaRulesWarning("java_library"),
	"native-java-plugin":                 NativeJavaRulesWarning("java_plugin"),
	"native-java-test":                   NativeJavaRulesWarning("java_test"),
	"native-java-package-config":         NativeJavaToolchainRulesWarning("java_package_configuration"),
	"native-java-runtime":                NativeJavaToolchainRulesWarning("java_runtime"),
	"native-java-toolchain":              NativeJavaToolchainRulesWarning("java_toolchain"),
	"native-java-common":                 NativeJavaSymbolsWarning("java_common", "java_common"),
	"native-java-info":                   NativeJavaSymbolsWarning("JavaInfo", "java_info"),
	"native-java-plugin-info":            NativeJavaSymbolsWarning("JavaPluginInfo", "java_plugin_info"),
	"native-proto":                       NativeProtoRulesWarning("proto_library"),
	"native-java-proto":                  NativeProtoRulesWarning("java_proto_library"),
	"native-java-lite-proto":             NativeProtoRulesWarning("java_lite_proto_library"),
	"native-cc-proto":                    NativeProtoRulesWarning("cc_proto_library"),
	"native-proto-lang-toolchain":        nativeProtoLangToolchainWarning,
	"native-proto-info":                  nativeProtoSymbolsWarning("ProtoInfo", "proto_info.bzl"),
	"native-proto-common":                nativeProtoSymbolsWarning("proto_common", "proto_common.bzl"),
	"native-proto-lang-toolchain-info":   nativeProtoSymbolsWarning("ProtoLangToolchainInfo", "proto_lang_toolchain_info.bzl"),
	"native-py":                          nativePyRulesWarning,
	"native-sh-binary":                   NativeShellRulesWarning("sh_binary"),
	"native-sh-library":                  NativeShellRulesWarning("sh_library"),
	"native-sh-test":                     NativeShellRulesWarning("sh_test"),
	"positional-args":                    positionalArgumentsWarning,
	"unnamed-macro":                      unnamedMacroWarning,
}

// nonDefaultWarnings contains warnings that are enabled by default because they're not applicable
// for all files and cause too much diff noise when applied.
var nonDefaultWarnings = map[string]bool{
	"unsorted-dict-items": true, // dict items should be sorted
}

// fileWarningWrapper is a wrapper that converts a file warning function to a generic function.
// A generic function takes a `pkg string` and a `*ReadFile` arguments which are not used for file warnings,
// so they are just removed.
func fileWarningWrapper(fct func(f *build.File) []*LinterFinding) func(*build.File, string, *FileReader) []*LinterFinding {
	return func(f *build.File, _ string, _ *FileReader) []*LinterFinding {
		return fct(f)
	}
}

// multiFileWarningWrapper is a wrapper that converts a multifile warning function to a generic function.
// A generic function takes a `pkg string` argument which is not used for file warnings, so it's just removed.
func multiFileWarningWrapper(fct func(f *build.File, fileReader *FileReader) []*LinterFinding) func(*build.File, string, *FileReader) []*LinterFinding {
	return func(f *build.File, _ string, fileReader *FileReader) []*LinterFinding {
		return fct(f, fileReader)
	}
}

// ruleWarningWrapper is a wrapper that converts a per-rule function to a per-file function.
// It also doesn't run on .bzl or default files, only on BUILD and WORKSPACE files.
func ruleWarningWrapper(ruleWarning func(call *build.CallExpr, pkg string) *LinterFinding) func(*build.File, string, *FileReader) []*LinterFinding {
	return func(f *build.File, pkg string, _ *FileReader) []*LinterFinding {
		if f.Type != build.TypeBuild {
			return nil
		}
		var findings []*LinterFinding
		for _, stmt := range f.Stmt {
			switch stmt := stmt.(type) {
			case *build.CallExpr:
				finding := ruleWarning(stmt, pkg)
				if finding != nil {
					findings = append(findings, finding)
				}
			case *build.Comprehension:
				// Rules are often called within list comprehensions, e.g. [my_rule(foo) for foo in bar]
				if call, ok := stmt.Body.(*build.CallExpr); ok {
					finding := ruleWarning(call, pkg)
					if finding != nil {
						findings = append(findings, finding)
					}
				}
			}
		}
		return findings
	}
}

// runWarningsFunction runs a linter/fixer function over a file and applies the fixes conditionally
func runWarningsFunction(category string, f *build.File, fct func(f *build.File, pkg string, fileReader *FileReader) []*LinterFinding, formatted *[]byte, mode LintMode, fileReader *FileReader) []*Finding {
	findings := []*Finding{}
	for _, w := range fct(f, f.Pkg, fileReader) {
		if !DisabledWarning(f, w.Start.Line, category) {
			finding := makeFinding(f, w.Start, w.End, category, w.URL, w.Message, true, len(w.Replacement) > 0, nil)
			if len(w.Replacement) > 0 {
				// An automatic fix exists
				switch mode {
				case ModeFix:
					// Apply the fix and discard the finding
					for _, r := range w.Replacement {
						*r.Old = r.New
					}
					finding = nil
				case ModeSuggest:
					// Apply the fix, calculate the diff and roll back the fix
					newContents := formatWithFix(f, &w.Replacement)

					start, end, replacement := calculateDifference(formatted, &newContents)
					finding.Replacement = &Replacement{
						Description: w.Message,
						Start:       start,
						End:         end,
						Content:     replacement,
					}
				}
			}
			if finding != nil {
				findings = append(findings, finding)
			}
		}
	}
	return findings
}

// HasDisablingComment checks if a node has a comment that disables a certain warning
func HasDisablingComment(expr build.Expr, warning string) bool {
	return edit.ContainsDisableComment(expr, warning)
}

// DisabledWarning checks if the warning was disabled by a comment.
// The comment format is buildozer: disable=<warning>
func DisabledWarning(f *build.File, findingLine int, warning string) bool {
	disabled := false

	build.Walk(f, func(expr build.Expr, stack []build.Expr) {
		if expr == nil {
			return
		}

		start, end := expr.Span()
		comments := expr.Comment()
		if len(comments.Before) > 0 {
			start, _ = comments.Before[0].Span()
		}
		if len(comments.After) > 0 {
			_, end = comments.After[len(comments.After)-1].Span()
		}
		if findingLine < start.Line || findingLine > end.Line {
			return
		}

		if HasDisablingComment(expr, warning) {
			disabled = true
			return
		}
	})

	return disabled
}

// FileWarnings returns a list of all warnings found in the file.
func FileWarnings(f *build.File, enabledWarnings []string, formatted *[]byte, mode LintMode, fileReader *FileReader) []*Finding {
	findings := []*Finding{}

	// Sort the warnings to make sure they're applied in the same determined order
	// Make a local copy first to avoid race conditions
	warnings := append([]string{}, enabledWarnings...)
	sort.Strings(warnings)

	// If suggestions are requested and formatted file is not provided, format it to compare modified versions with
	if mode == ModeSuggest && formatted == nil {
		contents := build.Format(f)
		formatted = &contents
	}

	for _, warn := range warnings {
		if fct, ok := FileWarningMap[warn]; ok {
			findings = append(findings, runWarningsFunction(warn, f, fileWarningWrapper(fct), formatted, mode, fileReader)...)
		} else if fct, ok := MultiFileWarningMap[warn]; ok {
			findings = append(findings, runWarningsFunction(warn, f, multiFileWarningWrapper(fct), formatted, mode, fileReader)...)
		} else if fct, ok := RuleWarningMap[warn]; ok {
			findings = append(findings, runWarningsFunction(warn, f, ruleWarningWrapper(fct), formatted, mode, fileReader)...)
		} else {
			log.Printf("unexpected warning %q", warn)
			os.Exit(1)
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Start.Line != findings[j].Start.Line {
			return findings[i].Start.Line < findings[j].Start.Line
		}
		if findings[i].Start.LineRune != findings[j].Start.LineRune {
			return findings[i].Start.LineRune < findings[j].Start.LineRune
		}
		if findings[i].Category != findings[j].Category {
			return findings[i].Category < findings[j].Category
		}
		return findings[i].Message < findings[j].Message
	})
	return findings
}

// formatWithFix applies a fix, formats a file, and rolls back the fix
func formatWithFix(f *build.File, replacements *[]LinterReplacement) []byte {
	for i := range *replacements {
		r := (*replacements)[i]
		old := *r.Old
		*r.Old = r.New
		defer func() { *r.Old = old }()
	}

	return build.Format(f)
}

// calculateDifference compares two file contents and returns a replacement in the form of
// a 3-tuple (byte from, byte to (non inclusive), a string to replace with).
func calculateDifference(old, new *[]byte) (start, end int, replacement string) {
	commonPrefix := 0 // length of the common prefix
	for i, b := range *old {
		if i >= len(*new) || b != (*new)[i] {
			break
		}
		commonPrefix++
	}

	commonSuffix := 0 // length of the common suffix
	for i := range *old {
		b := (*old)[len(*old)-1-i]
		if i >= len(*new) || b != (*new)[len(*new)-1-i] {
			break
		}
		commonSuffix++
	}

	// In some cases common suffix and prefix can overlap. E.g. consider the following case:
	//   old = "abc"
	//   new = "abdbc"
	// In this case the common prefix is "ab" and the common suffix is "bc".
	// If they overlap, just shorten the suffix so that they don't.
	// The new suffix will be just "c".
	if commonPrefix+commonSuffix > len(*old) {
		commonSuffix = len(*old) - commonPrefix
	}
	if commonPrefix+commonSuffix > len(*new) {
		commonSuffix = len(*new) - commonPrefix
	}
	return commonPrefix, len(*old) - commonSuffix, string((*new)[commonPrefix:(len(*new) - commonSuffix)])
}

// FixWarnings fixes all warnings that can be fixed automatically.
func FixWarnings(f *build.File, enabledWarnings []string, verbose bool, fileReader *FileReader) {
	warnings := FileWarnings(f, enabledWarnings, nil, ModeFix, fileReader)
	if verbose {
		fmt.Fprintf(os.Stderr, "%s: applied fixes, %d warnings left\n",
			f.DisplayPath(),
			len(warnings))
	}
}

func collectAllWarnings() []string {
	var result []string
	// Collect list of all warnings.
	for k := range FileWarningMap {
		result = append(result, k)
	}
	for k := range MultiFileWarningMap {
		result = append(result, k)
	}
	for k := range RuleWarningMap {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// AllWarnings is the list of all available warnings.
var AllWarnings = collectAllWarnings()

func collectDefaultWarnings() []string {
	warnings := []string{}
	for _, warning := range AllWarnings {
		if !nonDefaultWarnings[warning] {
			warnings = append(warnings, warning)
		}
	}
	return warnings
}

// DefaultWarnings is the list of all warnings that should be used inside google3
var DefaultWarnings = collectDefaultWarnings()
