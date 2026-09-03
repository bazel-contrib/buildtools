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

// Buildifier, a tool to parse and format BUILD files.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/buildifier/config"
	"github.com/bazelbuild/buildtools/buildifier/utils"
	"github.com/bazelbuild/buildtools/differ"
	"github.com/bazelbuild/buildtools/file"
	"github.com/bazelbuild/buildtools/wspace"
)

var buildVersion = "redacted"
var buildScmRevision = "redacted"

func usage() {
	fmt.Fprintf(flag.CommandLine.Output(), `usage: buildifier [-d] [-v] [-r] [-config=path.json] [-diff_command=command] [-help] [-multi_diff] [-mode=mode] [-lint=lint_mode] [-path=path] [files...]

Buildifier applies standard formatting to the named Starlark files.  The mode
flag selects the processing: check, diff, fix, or print_if_changed.  In check
mode, buildifier prints a list of files that need reformatting.  In diff mode,
buildifier shows the diffs that it would make.  It creates the diffs by running
a diff command, which can be specified using the -diff_command flag. You can
indicate that the diff command can show differences between more than two files
in the manner of tkdiff by specifying the -multi_diff flag.  In fix mode,
buildifier updates the files that need reformatting and, if the -v flag is
given, prints their names to standard error.  In print_if_changed mode,
buildifier shows the file contents it would write.  The default mode is fix. -d
is an alias for -mode=diff.

The lint flag selects the lint mode to be used: off, warn, fix.
In off mode, the linting is not performed.
In warn mode, buildifier prints warnings for common mistakes and suboptimal
coding practices that include links providing more context and fix suggestions.
In fix mode, buildifier updates the files with all warning resolutions produced
by automated fixes.
The default lint mode is off.

If no files are listed, buildifier reads a Starlark file from standard
input. In fix mode, it writes the reformatted Starlark file to standard output,
even if no changes are necessary.

Buildifier's reformatting depends in part on the path to the file relative
to the workspace directory. Normally buildifier deduces that path from the
file names given, but the path can be given explicitly with the -path
argument. This is especially useful when reformatting standard input,
or in scripts that reformat a temporary copy of a file.

Return codes used by buildifier:

  0: success, everything went well
  1: syntax errors in input
  2: usage errors: invoked incorrectly
  3: unexpected runtime errors: file I/O problems or internal bugs
  4: check mode failed (reformat is needed)

Full list of flags with their defaults:
`)
	flag.PrintDefaults()

	fmt.Fprintf(flag.CommandLine.Output(), `
Buildifier can also be configured via a JSON file.  The location of the file
is given by the -config flag, the BUILDIFIER_CONFIG environment variable, or
a file named '.buildifier.json' at the root of the workspace (e.g., in the same
directory as the WORKSPACE file).  The PWD environment variable or process
working directory is used to help find the workspace root.  If present, the file
is loaded into memory and becomes the base configuration that command line flags
override.  A sample configuration file can be printed to stdout by running
buildifier -config=example. The config file feature can be disabled completely
with -config=off.
`)
}

func main() {
	c := config.New()

	flags := c.FlagSet("buildifier", flag.ExitOnError)
	flag.CommandLine = flags
	flag.Usage = usage
	flags.Parse(os.Args[1:])
	args := flags.Args()

	if c.Help {
		flag.CommandLine.SetOutput(os.Stdout)
		usage()
		fmt.Println()
		os.Exit(0)
	}

	if c.Version {
		fmt.Printf("buildifier version: %s \n", buildVersion)
		fmt.Printf("buildifier scm revision: %s \n", buildScmRevision)
		os.Exit(0)
	}

	if c.ConfigPath == "" {
		c.ConfigPath = config.FindConfigPath("")
	}
	if c.ConfigPath != "" {
		if c.ConfigPath == "example" {
			fmt.Println(config.Example().String())
			os.Exit(0)
		}
		if c.ConfigPath != "off" {
			if err := c.LoadFile(); err != nil {
				fmt.Fprintf(os.Stderr, "buildifier: %s\n", err)
				os.Exit(2)
			}
			// re-parse with new possibly new defaults
			flags = c.FlagSet("buildifier", flag.ExitOnError)
			flag.CommandLine = flags
			flag.Usage = usage
			flags.Parse(os.Args[1:])
		}
	}

	if err := c.Validate(args); err != nil {
		fmt.Fprintf(os.Stderr, "buildifier: %s\n", err)
		os.Exit(2)
	}

	// Pass down debug flags into build package
	build.DisableRewrites = c.DisableRewrites
	build.AllowSort = c.AllowSort
	file.DisableSymlinkSafety = c.DisableSymlinkSafety

	differ, deprecationWarning := differ.Find()
	if c.DiffCommand != "" {
		differ.Cmd = c.DiffCommand
		differ.MultiDiff = c.MultiDiff
	} else {
		if deprecationWarning && c.Mode == "diff" {
			fmt.Fprintf(os.Stderr, "buildifier: selecting diff program with the BUILDIFIER_DIFF, BUILDIFIER_MULTIDIFF, and DISPLAY environment variables is deprecated, use flags -diff_command and -multi_diff instead\n")
		}
	}

	b := buildifier{c, differ}
	exitCode := b.run(args)

	os.Exit(exitCode)
}

type buildifier struct {
	config *config.Config
	differ *differ.Differ
}

func (b *buildifier) run(args []string) int {
	tf := &utils.TempFile{}
	defer tf.Clean()

	exitCode := 0
	var diagnostics *utils.Diagnostics
	if len(args) == 0 || (len(args) == 1 && (args)[0] == "-") {
		// Read from stdin, write to stdout.
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "buildifier: reading stdin: %v\n", err)
			return 2
		}
		if b.config.Mode == "fix" {
			b.config.Mode = "pipe"
		}
		var fileDiagnostics *utils.FileDiagnostics
		fileDiagnostics, exitCode = b.processFile("", data, false, tf)
		diagnostics = utils.NewDiagnostics(fileDiagnostics)
	} else {
		files := args
		if b.config.Recursive {
			var err error
			files, err = utils.ExpandDirectories(&args)
			if err != nil {
				fmt.Fprintf(os.Stderr, "buildifier: %v\n", err)
				return 3
			}
		}
		diagnostics, exitCode = b.processFiles(files, tf)
	}

	diagnosticsOutput := diagnostics.Format(b.config.Format, b.config.Verbose)
	if b.config.Format != "" {
		// Explicitly provided --format means the diagnostics are printed to stdout
		fmt.Print(diagnosticsOutput)
		// Exit code should be set to 0 so that other tools know they can safely parse the json
		exitCode = 0
	} else {
		// --format is not provided, stdout is reserved for file contents
		fmt.Fprint(os.Stderr, diagnosticsOutput)
	}

	if err := b.differ.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return 2
	}

	return exitCode
}

func (b *buildifier) processFiles(files []string, tf *utils.TempFile) (*utils.Diagnostics, int) {
	// Decide how many file reads to run in parallel.
	// At most 100, and at most one per 10 input files.
	nworker := 100
	if n := (len(files) + 9) / 10; nworker > n {
		nworker = n
	}
	runtime.GOMAXPROCS(nworker + 1)

	// Start nworker workers reading stripes of the input
	// argument list and sending the resulting data on
	// separate channels. file[k] is read by worker k%nworker
	// and delivered on ch[k%nworker].
	type result struct {
		file string
		data []byte
		err  error
	}

	ch := make([]chan result, nworker)
	for i := 0; i < nworker; i++ {
		ch[i] = make(chan result, 1)
		go func(i int) {
			for j := i; j < len(files); j += nworker {
				file := files[j]
				data, err := os.ReadFile(file)
				ch[i] <- result{file, data, err}
			}
		}(i)
	}

	exitCode := 0
	fileDiagnostics := []*utils.FileDiagnostics{}

	// Process files. The processing still runs in a single goroutine
	// in sequence. Only the reading of the files has been parallelized.
	// The goal is to optimize for runs where most files are already
	// formatted correctly, so that reading is the bulk of the I/O.
	for i, file := range files {
		res := <-ch[i%nworker]
		if res.file != file {
			fmt.Fprintf(os.Stderr, "buildifier: internal phase error: got %s for %s", res.file, file)
			os.Exit(3)
		}
		if res.err != nil {
			fmt.Fprintf(os.Stderr, "buildifier: %v\n", res.err)
			exitCode = 3
			continue
		}
		fd, newExitCode := b.processFile(file, res.data, len(files) > 1, tf)
		if fd != nil {
			fileDiagnostics = append(fileDiagnostics, fd)
		}
		if newExitCode != 0 {
			exitCode = newExitCode
		}
	}
	return utils.NewDiagnostics(fileDiagnostics...), exitCode
}

// processFile processes a single file containing data.
// It has been read from filename and should be written back if fixing.
func (b *buildifier) processFile(filename string, data []byte, displayFileNames bool, tf *utils.TempFile) (*utils.FileDiagnostics, int) {
	var exitCode int

	displayFilename := filename
	if b.config.WorkspaceRelativePath != "" {
		displayFilename = b.config.WorkspaceRelativePath
	}

	parser := utils.GetParser(b.config.InputType)

	f, err := parser(displayFilename, data)
	if err != nil {
		// Do not use buildifier: prefix on this error.
		// Since it is a parse error, it begins with file:line:
		// and we want that to be the first thing in the error.
		fmt.Fprintf(os.Stderr, "%v\n", err)
		if exitCode < 1 {
			exitCode = 1
		}
		return utils.InvalidFileDiagnostics(displayFilename), exitCode
	}

	if absoluteFilename, err := filepath.Abs(displayFilename); err == nil {
		f.WorkspaceRoot, f.Pkg, f.Label = wspace.SplitFilePath(absoluteFilename)
	}

	warnings := utils.Lint(f, b.config.Lint, &b.config.LintWarnings, b.config.Verbose)
	if len(warnings) > 0 {
		exitCode = 4
	}
	fileDiagnostics := utils.NewFileDiagnostics(f.DisplayPath(), warnings)

	ndata := build.Format(f)

	switch b.config.Mode {
	case "check":
		// check mode: print names of files that need formatting.
		if !bytes.Equal(data, ndata) {
			fileDiagnostics.Formatted = false
			return fileDiagnostics, 4
		}

	case "diff":
		// diff mode: run diff on old and new.
		if bytes.Equal(data, ndata) {
			return fileDiagnostics, exitCode
		}
		outfile, err := tf.WriteTemp(ndata)
		if err != nil {
			fmt.Fprintf(os.Stderr, "buildifier: %v\n", err)
			return fileDiagnostics, 3
		}
		infile := filename
		if filename == "" {
			// data was read from standard filename.
			// Write it to a temporary file so diff can read it.
			infile, err = tf.WriteTemp(data)
			if err != nil {
				fmt.Fprintf(os.Stderr, "buildifier: %v\n", err)
				return fileDiagnostics, 3
			}
		}
		if displayFileNames {
			fmt.Fprintf(os.Stderr, "%v:\n", f.DisplayPath())
		}
		if err := b.differ.Show(infile, outfile); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			return fileDiagnostics, 4
		}

	case "pipe":
		// pipe mode - reading from stdin, writing to stdout.
		// ("pipe" is not from the command line; it is set above in main.)
		os.Stdout.Write(ndata)

	case "fix":
		// fix mode: update files in place as needed.
		if bytes.Equal(data, ndata) {
			return fileDiagnostics, exitCode
		}

		err := file.WriteFileMode(filename, ndata, 0666)
		if err != nil {
			fmt.Fprintf(os.Stderr, "buildifier: %s\n", err)
			return fileDiagnostics, 3
		}

		if b.config.Verbose {
			fmt.Fprintf(os.Stderr, "fixed %s\n", f.DisplayPath())
		}
	case "print_if_changed":
		if bytes.Equal(data, ndata) {
			return fileDiagnostics, exitCode
		}

		if _, err := os.Stdout.Write(ndata); err != nil {
			fmt.Fprintf(os.Stderr, "buildifier: error writing output: %v\n", err)
			return fileDiagnostics, 3
		}
	}
	return fileDiagnostics, exitCode
}
