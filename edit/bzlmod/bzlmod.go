/*
Copyright 2023 Google LLC

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

// Package bzlmod contains functions for working with MODULE.bazel files.
package bzlmod

import (
	"path"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/labels"
)

// Proxies returns the names of extension proxies (i.e. the names of variables to which the result
// of a use_extension call is assigned) for the given extension with the given value of the
// dev_dependency attribute.
// Extension proxies created with "isolate = True" are ignored.
func Proxies(f *build.File, rawExtBzlFile string, extName string, dev bool) []string {
	apparentModuleName := getApparentModuleName(f)
	extBzlFile := normalizeLabelString(rawExtBzlFile, apparentModuleName)

	var proxies []string
	for _, stmt := range f.Stmt {
		proxy, rawBzlFile, name, isDev, isIsolated := parseUseExtension(stmt)
		if proxy == "" || isDev != dev || isIsolated {
			continue
		}
		bzlFile := normalizeLabelString(rawBzlFile, apparentModuleName)
		if bzlFile == extBzlFile && name == extName {
			proxies = append(proxies, proxy)
		}
	}

	return proxies
}

// AllProxies returns the names of all extension proxies (i.e. the names of variables to which the
// result of a use_extension call is assigned) corresponding to the same extension usage as the
// given proxy.
// For an isolated extension usage, a list containing only the given proxy is returned.
// For a non-isolated extension usage, the proxies of all non-isolated extension usages of the same
// extension with the same value for the dev_dependency parameter are returned.
// If the given proxy is not an extension proxy, nil is returned.
func AllProxies(f *build.File, proxy string) []string {
	for _, stmt := range f.Stmt {
		proxyCandidate, rawBzlFile, name, isDev, isIsolated := parseUseExtension(stmt)
		if proxyCandidate == proxy {
			if isIsolated {
				return []string{proxy}
			}
			return Proxies(f, rawBzlFile, name, isDev)
		}
	}
	return nil
}

// UseRepos returns the use_repo calls that use the given proxies.
func UseRepos(f *build.File, proxies []string) []*build.CallExpr {
	proxiesSet := make(map[string]struct{})
	for _, p := range proxies {
		proxiesSet[p] = struct{}{}
	}

	var useRepos []*build.CallExpr
	for _, stmt := range f.Stmt {
		if _, ok := stmt.(*build.CallExpr); !ok {
			continue
		}
		call := stmt.(*build.CallExpr)
		if _, ok := call.X.(*build.Ident); !ok {
			continue
		}
		if call.X.(*build.Ident).Name != "use_repo" || len(call.List) < 1 {
			continue
		}
		proxy, ok := call.List[0].(*build.Ident)
		if !ok {
			continue
		}
		if _, ok := proxiesSet[proxy.Name]; !ok {
			continue
		}
		useRepos = append(useRepos, call)
	}

	return useRepos
}

// NewUseRepo inserts and returns a new use_repo call after the last usage of any of the given
// proxies, where a usage is either a use_extension call or a tag definition.
func NewUseRepo(f *build.File, proxies []string) (*build.File, *build.CallExpr) {
	lastUsage, proxy := lastProxyUsage(f, proxies)
	if lastUsage == -1 {
		return f, nil
	}

	useRepo := &build.CallExpr{
		X: &build.Ident{Name: "use_repo"},
		List: []build.Expr{
			&build.Ident{Name: proxy},
		},
	}
	stmt := append(f.Stmt[:lastUsage+1], append([]build.Expr{useRepo}, f.Stmt[lastUsage+1:]...)...)

	return &build.File{Path: f.Path, Comments: f.Comments, Stmt: stmt, Type: build.TypeModule}, useRepo
}

// AddRepoUsages adds the given repos to the given use_repo calls without introducing duplicate
// arguments.
// useRepos must not be empty.
// Keyword arguments are preserved but adding them is currently not supported.
func AddRepoUsages(useRepos []*build.CallExpr, repos ...string) {
	if len(repos) == 0 {
		return
	}
	if len(useRepos) == 0 {
		panic("useRepos must not be empty")
	}

	seen := make(map[string]struct{})
	for _, useRepo := range useRepos {
		if len(useRepo.List) == 0 {
			// Invalid use_repo call, skip.
			continue
		}
		for _, arg := range useRepo.List[1:] {
			seen[repoFromUseRepoArg(arg)] = struct{}{}
		}
	}

	lastUseRepo := getLastUseRepo(useRepos)
	for _, repo := range repos {
		if _, ok := seen[repo]; ok {
			continue
		}
		// Sorting of use_repo arguments is handled by Buildify.
		// TODO: Add a keyword argument instead if repo is of the form "key=value".
		lastUseRepo.List = append(lastUseRepo.List, &build.StringExpr{Value: repo})
	}
}

// RemoveRepoUsages removes the given repos from the given use_repo calls.
// Repositories are identified via their names as exported by the module extension (i.e. the value
// rather than the key in the case of keyword arguments).
func RemoveRepoUsages(useRepos []*build.CallExpr, repos ...string) {
	if len(useRepos) == 0 || len(repos) == 0 {
		return
	}

	toRemove := make(map[string]struct{})
	for _, repo := range repos {
		toRemove[repo] = struct{}{}
	}

	for _, useRepo := range useRepos {
		if len(useRepo.List) == 0 {
			// Invalid use_repo call, skip.
			continue
		}
		var args []build.Expr
		// Skip over ext in use_repo(ext, ...).
		for _, arg := range useRepo.List[1:] {
			repo := repoFromUseRepoArg(arg)
			if _, remove := toRemove[repo]; !remove {
				args = append(args, arg)
			}
		}
		useRepo.List = append(useRepo.List[:1], args...)
	}
}

func getLastUseRepo(useRepos []*build.CallExpr) *build.CallExpr {
	var lastUseRepo *build.CallExpr
	for _, useRepo := range useRepos {
		if lastUseRepo == nil || useRepo.Pos.Byte > lastUseRepo.Pos.Byte {
			lastUseRepo = useRepo
		}
	}
	return lastUseRepo
}

// repoFromUseRepoArg returns the repository name used by the module extension itself from a
// use_repo argument.
func repoFromUseRepoArg(arg build.Expr) string {
	switch arg := arg.(type) {
	case *build.StringExpr:
		// use_repo(ext, "repo") --> repo
		return arg.Value
	case *build.AssignExpr:
		// use_repo(ext, my_repo = "repo") --> repo
		if repo, ok := arg.RHS.(*build.StringExpr); ok {
			return repo.Value
		}
	}
	return ""
}

// getApparentModuleName returns the apparent name used for the repository of the module defined
// in the given MODULE.bazel file.
func getApparentModuleName(f *build.File) string {
	apparentName := ""

	for _, module := range f.Rules("module") {
		if repoName := module.AttrString("repo_name"); repoName != "" {
			apparentName = repoName
		} else if name := module.AttrString("name"); name != "" {
			apparentName = name
		}
	}

	return apparentName
}

// normalizeLabelString converts a label string into the form @apparent_name//path/to:target.
func normalizeLabelString(rawLabel, apparentModuleName string) string {
	label := labels.ParseRelative(rawLabel, "")
	if label.Repository == "" {
		// This branch is taken in two different cases:
		// 1. The label is relative. In this case, labels.ParseRelative populates the Package field
		//    but not the Repository field.
		// 2. The label is of the form "@//pkg:extension.bzl". Normalize to spelling out the
		//    apparent name of the root module. Note that this syntax is only allowed in the root
		//    module, but since we are inspecting its module file as a tool, we can assume that the
		//    current module is the root module.
		label.Repository = apparentModuleName
	}
	return label.Format()
}

func parseUseExtension(stmt build.Expr) (proxy string, bzlFile string, name string, dev bool, isolate bool) {
	assign, ok := stmt.(*build.AssignExpr)
	if !ok {
		return
	}
	lhsIdent, ok := assign.LHSIdent()
	if !ok {
		return
	}
	if _, ok = assign.RHS.(*build.CallExpr); !ok {
		return
	}
	call := assign.RHS.(*build.CallExpr)
	if _, ok = call.X.(*build.Ident); !ok {
		return
	}
	if call.X.(*build.Ident).Name != "use_extension" {
		return
	}
	if len(call.List) < 2 {
		// Missing required positional arguments.
		return
	}
	bzlFileExpr, ok := call.List[0].(*build.StringExpr)
	if !ok {
		return
	}
	nameExpr, ok := call.List[1].(*build.StringExpr)
	if !ok {
		return
	}
	// Check for the optional dev_dependency keyword argument.
	if len(call.List) > 2 {
		for _, arg := range call.List[2:] {
			dev = dev || parseBooleanKeywordArg(arg, "dev_dependency")
			isolate = isolate || parseBooleanKeywordArg(arg, "isolate")
		}
	}
	return lhsIdent.Name, bzlFileExpr.Value, nameExpr.Value, dev, isolate
}

// parseBooleanKeywordArg parses a keyword argument of type bool that is assumed to default to
// False.
func parseBooleanKeywordArg(arg build.Expr, name string) bool {
	keywordArg, ok := arg.(*build.AssignExpr)
	if !ok {
		return false
	}
	argName, ok := keywordArg.LHS.(*build.Ident)
	if !ok || argName.Name != name {
		return false
	}
	argValue, ok := keywordArg.RHS.(*build.Ident)
	// We assume that any expression other than "False" evaluates to True as otherwise there would
	// be no reason to specify the argument - MODULE.bazel files are entirely static with no
	// external inputs, so every expression always evaluates to the same value.
	if ok && argValue.Name == "False" {
		return false
	}
	return true
}

func parseTag(stmt build.Expr) string {
	call, ok := stmt.(*build.CallExpr)
	if !ok {
		return ""
	}
	if _, ok := call.X.(*build.DotExpr); !ok {
		return ""
	}
	dot := call.X.(*build.DotExpr)
	if _, ok := dot.X.(*build.Ident); !ok {
		return ""
	}
	return dot.X.(*build.Ident).Name
}

// lastProxyUsage returns the index of the last statement in the given file that uses one of the
// given extension proxies (either in a use_extension assignment or tag call). If no such statement
// exists, -1 is returned.
func lastProxyUsage(f *build.File, proxies []string) (lastUsage int, lastProxy string) {
	proxiesSet := make(map[string]struct{})
	for _, p := range proxies {
		proxiesSet[p] = struct{}{}
	}

	lastUsage = -1
	for i, stmt := range f.Stmt {
		proxy, _, _, _, _ := parseUseExtension(stmt)
		if proxy != "" {
			_, isUsage := proxiesSet[proxy]
			if isUsage {
				lastUsage = i
				lastProxy = proxy
				continue
			}
		}

		proxy = parseTag(stmt)
		if proxy != "" {
			_, isUsage := proxiesSet[proxy]
			if isUsage {
				lastUsage = i
				lastProxy = proxy
				continue
			}
		}
	}

	return lastUsage, lastProxy
}

// ExtractModuleToApparentNameMapping collects the mapping of module names (e.g. "rules_go") to
// user-configured apparent names (e.g. "my_rules_go") from the repo's MODULE.bazel, if it exists.
// The given function is called with a repo-relative, slash-separated path and should return the
// content of the MODULE.bazel or *.MODULE.bazel file at that path, or nil if the file does not
// exist.
// See https://bazel.build/external/module#repository_names_and_strict_deps for more information on
// apparent names.
func ExtractModuleToApparentNameMapping(fileReader func(relPath string) *build.File) func(string) string {
	moduleToApparentName := collectApparentNames(fileReader, "MODULE.bazel")

	return func(moduleName string) string {
		return moduleToApparentName[moduleName]
	}
}

// Collects the mapping of module names (e.g. "rules_go") to user-configured apparent names (e.g.
// "my_rules_go"). See https://bazel.build/external/module#repository_names_and_strict_deps for more
// information on apparent names.
func collectApparentNames(fileReader func(relPath string) *build.File, relPath string) map[string]string {
	apparentNames := make(map[string]string)
	seenFiles := make(map[string]struct{})
	filesToProcess := []string{relPath}

	for len(filesToProcess) > 0 {
		f := filesToProcess[0]
		filesToProcess = filesToProcess[1:]
		if _, seen := seenFiles[f]; seen {
			continue
		}
		seenFiles[f] = struct{}{}
		bf := fileReader(f)
		if bf == nil {
			return nil
		}
		names, includeLabels := collectApparentNamesAndIncludes(bf)
		for name, apparentName := range names {
			apparentNames[name] = apparentName
		}
		for _, includeLabel := range includeLabels {
			l := labels.Parse(includeLabel)
			p := path.Join(l.Package, l.Target)
			filesToProcess = append(filesToProcess, p)
		}
	}

	return apparentNames
}

func collectApparentNamesAndIncludes(f *build.File) (map[string]string, []string) {
	apparentNames := make(map[string]string)
	var includeLabels []string

	for _, dep := range f.Rules("") {
		if dep.ExplicitName() == "" {
			if ident, ok := dep.Call.X.(*build.Ident); !ok || ident.Name != "include" {
				continue
			}
			if len(dep.Call.List) != 1 {
				continue
			}
			if str, ok := dep.Call.List[0].(*build.StringExpr); ok {
				includeLabels = append(includeLabels, str.Value)
			}
			continue
		}
		if dep.Kind() != "module" && dep.Kind() != "bazel_dep" {
			continue
		}
		// We support module in addition to bazel_dep to handle language repos that use Gazelle to
		// manage their own BUILD files.
		if name := dep.AttrString("name"); name != "" {
			if repoName := dep.AttrString("repo_name"); repoName != "" {
				apparentNames[name] = repoName
			} else {
				apparentNames[name] = name
			}
		}
	}

	return apparentNames, includeLabels
}
