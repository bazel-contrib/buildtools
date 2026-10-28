/*
Copyright 2017 Google LLC

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

// The unused_deps binary prints out buildozer commands for removing
// unused Java dependencies from java_library Bazel rules.
package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/config"
	depspb "github.com/bazel-contrib/buildtools/v10/deps_proto"
	"github.com/bazel-contrib/buildtools/v10/edit"
	eapb "github.com/bazel-contrib/buildtools/v10/extra_actions_base_proto"
	"github.com/bazel-contrib/buildtools/v10/file"
	"github.com/bazel-contrib/buildtools/v10/labels"
	"github.com/golang/protobuf/proto"
)

var (
	buildVersion     = "redacted"
	buildScmRevision = "redacted"

	version              = flag.Bool("version", false, "Print the version of unused_deps")
	cQuery               = flag.Bool("cquery", false, "Use 'cquery' command instead of 'query'")
	buildTool            = flag.String("build_tool", config.DefaultBuildTool, config.BuildToolHelp)
	extraActionFileName  = flag.String("extra_action_file", "", config.ExtraActionFileNameHelp)
	outputFileName       = flag.String("output_file", "", "used only with extra_action_file")
	buildOptions         = stringList("extra_build_flags", "Extra build flags to use when building the targets.")
	disableSymlinkSafety = flag.Bool("disable_symlink_safety", false, "per default unused_deps will not write to symlinks pointing outside of the Bazel workspace. Setting this to true will disable this behavior")

	blazeFlags = []string{"--tool_tag=unused_deps", "--keep_going", "--color=yes", "--curses=yes"}

	aspect = `
# Explicitly creates a params file for a Javac action.
def _javac_params(target, ctx):
    params = []
    for action in target.actions:
        if not action.mnemonic == "Javac" and not action.mnemonic == "KotlinCompile":
            continue
        output = ctx.actions.declare_file("%s.javac_params" % target.label.name)
        args = ctx.actions.args()
        args.add_all(action.argv)
        ctx.actions.write(
            output = output,
            content = args,
        )
        params.append(output)
        break
    return [OutputGroupInfo(unused_deps_outputs = depset(params))]

javac_params = aspect(
    implementation = _javac_params,
)
`
)

func stringList(name, help string) func() []string {
	f := flag.String(name, "", help)
	return func() []string {
		if *f == "" {
			return nil
		}
		res := strings.Split(*f, ",")
		for i := range res {
			res[i] = strings.TrimSpace(res[i])
		}
		return res
	}
}

// getJarPath prints the path to the output jar file specified in the extra_action file at path.
func getJarPath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	i := &eapb.ExtraActionInfo{}
	if err := proto.Unmarshal(data, i); err != nil {
		return "", err
	}
	ext, err := proto.GetExtension(i, eapb.E_JavaCompileInfo_JavaCompileInfo)
	if err != nil {
		return "", err
	}
	jci, ok := ext.(*eapb.JavaCompileInfo)
	if !ok {
		return "", errors.New("no JavaCompileInfo in " + path)
	}
	return jci.GetOutputjar(), nil
}

// writeUnusedDeps writes the labels of unused direct deps, one per line, to outputFileName.
func writeUnusedDeps(jarPath, outputFileName string) {
	depsPath := strings.Replace(jarPath, ".jar", ".jdeps", 1)
	paramsPath := jarPath + "-2.params"
	file, _ := os.Create(outputFileName)
	for dep := range unusedDeps(depsPath, directDepParams(paramsPath)) {
		file.WriteString(dep + "\n")
	}
}

func cmdWithStderr(name string, arg ...string) *exec.Cmd {
	cmd := exec.Command(name, arg...)
	cmd.Stderr = os.Stderr
	return cmd
}

// blazeInfo retrieves the blaze info value for a given key.
func blazeInfo(key string) (value string) {
	out, err := cmdWithStderr(*buildTool, "info", key).Output()
	if err != nil {
		log.Printf("'%s info %s' failed: %s", *buildTool, key, err)
	}
	return strings.TrimSpace(bytes.NewBuffer(out).String())
}

// inputFileName returns a blaze output file name from which to read input.
func inputFileName(blazeBin, pkg, ruleName, extension string) string {
	name := fmt.Sprintf("%s/%s/lib%s.%s", blazeBin, pkg, ruleName, extension) // *_library
	if _, err := os.Stat(name); err == nil {
		return name
	}
	// lazily let the caller handle it if this doesn't exist
	return fmt.Sprintf("%s/%s/%s.%s", blazeBin, pkg, ruleName, extension) // *_{binary,test}
}

// directDepParams reads the jar-2.params files, looking for a
// "--direct_dependencies" argument.  When found, the direct dependencies are
// returned as a map from jar file names to labels.
func directDepParams(blazeOutputPath string, paramsFileNames ...string) (depsByJar map[string]string) {
	depsByJar = make(map[string]string)
	errs := make([]error, 0)
	for _, paramsFileName := range paramsFileNames {
		data, err := os.ReadFile(paramsFileName)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		// The classpath param exceeds MaxScanTokenSize, so we scan just the
		// dependencies section.
		directDepsFlag := []byte("--direct_dependencies")
		arg := bytes.Index(data, directDepsFlag)
		if arg < 0 {
			continue
		}
		first := arg + len(directDepsFlag) + 1

		scanner := bufio.NewScanner(bytes.NewReader(data[first:]))
		for scanner.Scan() {
			jar := scanner.Text()
			if strings.HasPrefix(jar, "--") {
				break
			}
			label, err := jarManifestValue(blazeOutputPath+strings.TrimPrefix(jar, "bazel-out"), "Target-Label")
			if err != nil {
				continue
			}
			if strings.HasPrefix(label, "@@") || strings.HasPrefix(label, "@/") {
				label = label[1:]
			}
			depsByJar[jar] = label
		}
		if err := scanner.Err(); err != nil {
			log.Printf("reading %s: %s", paramsFileName, err)
		}
	}
	if len(errs) == len(paramsFileNames) {
		for _, err := range errs {
			log.Println(err)
		}
	}
	return depsByJar
}

// unusedDeps returns a set of labels that are unused deps.
// It reads Dependencies proto messages from depsFileName (a jdeps file), which indicate deps used
// at compile time, and returns those values in the depsByJar map that aren't used at compile time.
func unusedDeps(depsFileName string, depsByJar map[string]string) (unusedDeps map[string]bool) {
	unusedDeps = make(map[string]bool)
	data, err := os.ReadFile(depsFileName)
	if err != nil {
		log.Println(err)
		return unusedDeps
	}
	dependencies := &depspb.Dependencies{}
	if err := proto.Unmarshal(data, dependencies); err != nil {
		log.Println(err)
		return unusedDeps
	}
	for _, label := range depsByJar {
		unusedDeps[label] = true
	}
	for _, dependency := range dependencies.Dependency {
		if *dependency.Kind == depspb.Dependency_EXPLICIT {
			delete(unusedDeps, depsByJar[*dependency.Path])
		}
	}
	return unusedDeps
}

// parseBuildFile tries to read and parse the contents of buildFileName.
func parseBuildFile(buildFileName string) (buildFile *build.File, err error) {
	data, err := os.ReadFile(buildFileName)
	if err != nil {
		return nil, err
	}
	return build.Parse(buildFileName, data)
}

// getDepsExpr tries to parse the content of buildFileName and return the deps Expr for ruleName.
func getDepsExpr(buildFileName string, ruleName string) build.Expr {
	buildFile, err := parseBuildFile(buildFileName)
	if buildFile == nil {
		log.Printf("%s when parsing %s", err, buildFileName)
		return nil
	}
	rule := edit.FindRuleByName(buildFile, ruleName)
	if rule == nil {
		log.Printf("%s not found in %s", ruleName, buildFileName)
		return nil
	}
	depsExpr := rule.Attr("deps")
	if depsExpr == nil {
		log.Printf("no deps attribute for %s in %s", ruleName, buildFileName)
	}
	return depsExpr
}

// hasRuntimeComment returns true if expr has an EOL comment containing the word "runtime".
// TODO(bazel-team): delete when this comment convention is extinct
func hasRuntimeComment(expr build.Expr) bool {
	for _, comment := range expr.Comment().Suffix {
		if strings.Contains(strings.ToLower(comment.Token), "runtime") {
			return true
		}
	}
	return false
}

// printCommands prints, for each key in the deps map, a buildozer command
// to remove that entry from the deps attribute of the rule identified by label.
// Returns true if at least one command was printed, or false otherwise.
func printCommands(label string, deps map[string]bool) (anyCommandPrinted bool) {
	buildFileName, repo, pkg, ruleName := edit.InterpretLabelWithRepo(label)
	if repo != "" {
		outputBase := blazeInfo(config.DefaultOutputBase)
		buildFileName = fmt.Sprintf("%s/external/%s/%s", outputBase, repo, buildFileName)
	}

	depsExpr := getDepsExpr(buildFileName, ruleName)
	for _, li := range edit.AllLists(depsExpr) {
		for _, elem := range li.List {
			for dep := range deps {
				str, ok := elem.(*build.StringExpr)
				if !ok {
					continue
				}
				buildLabel := str.Value
				if repo != "" && buildLabel[:2] == "//" {
					buildLabel = fmt.Sprintf("@%s%s", repo, str.Value)
				}
				if !labels.Equal(buildLabel, dep, pkg) {
					continue
				}
				if hasRuntimeComment(str) {
					fmt.Printf("buildozer 'move deps runtime_deps %s' %s\n", str.Value, label)
				} else {
					// add dep's exported dependencies to label before removing dep
					fmt.Printf("buildozer \"add deps $(%s query 'labels(exports, %s)' | tr '\\n' ' ')\" %s\n", *buildTool, str.Value, label)
					fmt.Printf("buildozer 'remove deps %s' %s\n", str.Value, label)
				}
				anyCommandPrinted = true
			}
		}
	}
	return anyCommandPrinted
}

// setupAspect creates a workspace in a tmpdir and populates it with an aspect,
// which is used with --override_repository below.
func setupAspect() (string, error) {
	tmp, err := os.MkdirTemp(os.TempDir(), "unused_deps")
	if err != nil {
		return "", err
	}
	for _, f := range []string{"MODULE.bazel", "WORKSPACE", "BUILD"} {
		if err := file.WriteFileMode(path.Join(tmp, f), []byte{}, 0666); err != nil {
			return "", err
		}
	}
	if err := file.WriteFileMode(path.Join(tmp, "unused_deps.bzl"), []byte(aspect), 0666); err != nil {
		return "", err
	}
	return tmp, nil
}

func usage() {
	_, _ = fmt.Fprintf(flag.CommandLine.Output(), `usage: unused_deps TARGET...

For Java rules in TARGETs, prints commands to delete deps unused at compile time.
Note these may be used at run time; see documentation for more information.

`)
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	flag.Usage = usage
	flag.Parse()
	file.DisableSymlinkSafety = *disableSymlinkSafety
	if *version {
		fmt.Printf("unused_deps version: %s \n", buildVersion)
		fmt.Printf("unused_deps scm revision: %s \n", buildScmRevision)
		os.Exit(0)
	}

	if *extraActionFileName != "" {
		jarPath, err := getJarPath(*extraActionFileName)
		if err != nil {
			log.Fatal(err)
		}
		writeUnusedDeps(jarPath, *outputFileName)
		return
	}
	targetPatterns := flag.Args()
	if len(targetPatterns) == 0 {
		targetPatterns = []string{"//..."}
	}
	queryCmd := []string{}
	if *cQuery {
		queryCmd = append(queryCmd, "cquery")
	} else {
		queryCmd = append(queryCmd, "query")
	}
	queryCmd = append(queryCmd, blazeFlags...)
	queryCmd = append(
		queryCmd, fmt.Sprintf("kind('(kt|java|android)_*', %s)", strings.Join(targetPatterns, " + ")))

	log.Printf("running: %s %s", *buildTool, strings.Join(queryCmd, " "))
	queryOut, err := cmdWithStderr(*buildTool, queryCmd...).Output()
	if err != nil {
		log.Print(err)
	}
	if len(queryOut) == 0 {
		fmt.Fprintln(os.Stderr, "found no targets of kind (kt|java|android)_*")
		usage()
	}

	aspectDir, err := setupAspect()
	if err != nil {
		log.Print(err)
		os.Exit(1)
	}
	defer func() {
		os.RemoveAll(aspectDir)
	}()

	buildCmd := []string{"build"}
	buildCmd = append(buildCmd, blazeFlags...)
	buildCmd = append(buildCmd, config.DefaultExtraBuildFlags...)
	buildCmd = append(buildCmd, "--output_groups=+unused_deps_outputs")
	buildCmd = append(buildCmd, "--inject_repository=unused_deps="+aspectDir)
	buildCmd = append(buildCmd, "--aspects=@unused_deps//:unused_deps.bzl%javac_params")
	buildCmd = append(buildCmd, buildOptions()...)

	blazeArgs := append(buildCmd, targetPatterns...)

	log.Printf("running: %s %s", *buildTool, strings.Join(blazeArgs, " "))
	cmdWithStderr(*buildTool, blazeArgs...).Run()
	binDir := blazeInfo(config.DefaultBinDir)
	blazeOutputPath := blazeInfo(config.DefaultOutputPath)
	fmt.Fprintf(os.Stderr, "\n") // vertical space between build output and unused_deps output

	anyCommandPrinted := false
	for _, label := range strings.Fields(string(queryOut)) {
		if *cQuery && strings.HasPrefix(label, "(") {
			// cquery output includes the target's configuration ID.  Skip it.
			// https://docs.bazel.build/versions/main/cquery.html#configurations
			continue
		}
		_, repo, pkg, ruleName := edit.InterpretLabelWithRepo(label)
		blazeBin := binDir
		if repo != "" {
			blazeBin = fmt.Sprintf("%s/external/%s", binDir, repo)
		}
		depsByJar := directDepParams(blazeOutputPath, inputFileName(blazeBin, pkg, ruleName, "javac_params"))
		depsToRemove := unusedDeps(inputFileName(blazeBin, pkg, ruleName, "jdeps"), depsByJar)
		// TODO(bazel-team): instead of printing, have buildifier-like modes?
		anyCommandPrinted = printCommands(label, depsToRemove) || anyCommandPrinted
	}
	if !anyCommandPrinted {
		fmt.Fprintln(os.Stderr, "No unused deps found.")
	}
}
