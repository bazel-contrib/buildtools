/*
Copyright 2022 Google LLC

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

// Package config provides configuration objects for buildifier
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/bazelbuild/buildtools/tables"
	"github.com/bazelbuild/buildtools/warn"
	"github.com/bazelbuild/buildtools/wspace"
)

const buildifierJSONFilename = ".buildifier.json"

// New constructs a Config with default values.
func New() *Config {
	return &Config{
		InputType: "auto",
	}
}

// FindConfigPath locates the nearest buildifier configuration file.  First
// tries the value of the BUILDIFIER_CONFIG environment variable.  If no
// environment variable is defined, The configuration file will be resolved
// starting from the process cwd and searching up the file tree until a config
// file is (or isn't) found.
func FindConfigPath(rootDir string) string {
	if filename, ok := os.LookupEnv("BUILDIFIER_CONFIG"); ok {
		return filename
	}
	if rootDir == "" {
		rootDir, _ = os.Getwd() // best-effort, ignore error
	}
	dirname, err := wspace.Find(
		rootDir,
		map[string]func(os.FileInfo) bool{
			buildifierJSONFilename: func(fi os.FileInfo) bool {
				return fi.Mode()&os.ModeType == 0
			},
		},
	)
	if err != nil {
		return ""
	}
	return filepath.Join(dirname, buildifierJSONFilename)
}

// findTablesPath locates the specified table file starting from the process's
// current working directory. It searches upward through the directory tree
// until the file is found or the root of the workspace is reached.
func findTablesPath(file string) (string, error) {
	if wspace.IsRegularFile(file) {
		return file, nil
	}
	rootDir, _ := os.Getwd()
	dirname, err := wspace.Find(
		rootDir,
		map[string]func(os.FileInfo) bool{
			file: func(fi os.FileInfo) bool {
				return fi.Mode()&os.ModeType == 0
			},
		},
	)
	if err != nil {
		return file, err
	}
	return filepath.Join(dirname, file), nil
}

// Config is used to configure buildifier
type Config struct {
	// InputType determines the input file type: build (for BUILD files), bzl
	// (for .bzl files), workspace (for WORKSPACE files), default (for generic
	// Starlark files), module (for MODULE.bazel files)
	// or auto (default, based on the filename)
	InputType string `json:"type,omitempty"`
	// Format sets the diagnostics format: text or json (default text)
	Format string `json:"format,omitempty"`
	// Mode determines the formatting mode: check, diff, or fix (default fix)
	Mode string `json:"mode,omitempty"`
	// DiffMode is an alias for
	DiffMode bool `json:"diffMode,omitempty"`
	// Lint determines the lint mode: off, warn, or fix (default off)
	Lint string `json:"lint,omitempty"`
	// Warnings is a comma-separated list of warning identifiers used in the lint mode or "all"
	Warnings string `json:"warnings,omitempty"`
	// WarningsList is a list of warnings (alternative to comma-separated warnings string)
	WarningsList []string `json:"warningsList,omitempty"`
	// Recursive instructs buildifier to find starlark files recursively
	Recursive bool `json:"recursive,omitempty"`
	// Verbose instructs buildifier to output verbose diagnostics
	Verbose bool `json:"verbose,omitempty"`
	// DiffCommand is the command to run when the formatting mode is diff
	// (default uses the BUILDIFIER_DIFF, BUILDIFIER_MULTIDIFF, and DISPLAY
	// environment variables to create the diff command)
	DiffCommand string `json:"diffCommand,omitempty"`
	// MultiDiff means the command specified by the -diff_command flag can diff
	// multiple files in the style of tkdiff (default false)
	MultiDiff bool `json:"multiDiff,omitempty"`
	// TablesPath is the path to JSON file with custom table definitions that
	// will replace the built-in tables
	TablesPath string `json:"tables,omitempty"`
	// AddTablesPath path to JSON file with custom table definitions which will be merged with the built-in tables
	AddTablesPath string `json:"addTables,omitempty"`
	// WorkspaceRelativePath - assume BUILD file has this path relative to the workspace directory
	WorkspaceRelativePath string `json:"path,omitempty"`
	// DisableRewrites configures the list of buildifier rewrites to disable
	DisableRewrites ArrayFlags `json:"buildifier_disable,omitempty"`
	// AllowSort specifies additional sort contexts to treat as safe
	AllowSort ArrayFlags `json:"allowsort,omitempty"`

	// Help is true if the -h flag is set
	Help bool `json:"-"`
	// Version is true if the -v flag is set
	Version bool `json:"-"`
	// ConfigPath is the path to this config
	ConfigPath string `json:"-"`
	// LintWarnings is the final validated list of Lint/Fix warnings
	LintWarnings []string `json:"-"`
}

// LoadFile unmarshals JSON file from the ConfigPath field.
func (c *Config) LoadFile() error {
	file, err := os.Open(c.ConfigPath)
	if err != nil {
		return err
	}
	defer file.Close()
	return c.LoadReader(file)
}

// LoadReader unmarshals JSON data from the given reader. JSON with comments
// (JSONC) is supported.
func (c *Config) LoadReader(in io.Reader) error {
	data, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}
	data, err = parseJSONC(data)
	if err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return fmt.Errorf("unmarshaling config: %w", err)
	}
	return nil
}

// FlagSet returns a flag.FlagSet that can be used to override the config.
func (c *Config) FlagSet(name string, errorHandling flag.ErrorHandling) *flag.FlagSet {
	flags := flag.NewFlagSet(name, errorHandling)

	flags.BoolVar(&c.Help, "help", false, "print usage information")
	flags.BoolVar(&c.Version, "version", false, "print the version of buildifier")
	flags.BoolVar(&c.Verbose, "v", c.Verbose, "print verbose information to standard error")
	flags.BoolVar(&c.DiffMode, "d", c.DiffMode, "alias for -mode=diff")
	flags.BoolVar(&c.Recursive, "r", c.Recursive, "find starlark files recursively")
	flags.BoolVar(&c.MultiDiff, "multi_diff", c.MultiDiff, "the command specified by the -diff_command flag can diff multiple files in the style of tkdiff (default false)")
	flags.StringVar(&c.Mode, "mode", c.Mode, "formatting mode: check, diff, or fix (default fix)")
	flags.StringVar(&c.Format, "format", c.Format, "diagnostics format: text or json (default text)")
	flags.StringVar(&c.DiffCommand, "diff_command", c.DiffCommand, "command to run when the formatting mode is diff (default uses the BUILDIFIER_DIFF, BUILDIFIER_MULTIDIFF, and DISPLAY environment variables to create the diff command)")
	flags.StringVar(&c.Lint, "lint", c.Lint, "lint mode: off, warn, or fix (default off)")
	flags.StringVar(&c.Warnings, "warnings", c.Warnings, "comma-separated warnings used in the lint mode or \"all\"")
	flags.StringVar(&c.WorkspaceRelativePath, "path", c.WorkspaceRelativePath, "assume BUILD file has this path relative to the workspace directory")
	flags.StringVar(&c.TablesPath, "tables", c.TablesPath, "path to JSON file with custom table definitions which will replace the built-in tables")
	flags.StringVar(&c.AddTablesPath, "add_tables", c.AddTablesPath, "path to JSON file with custom table definitions which will be merged with the built-in tables")
	flags.StringVar(&c.InputType, "type", c.InputType, "Input file type: build (for BUILD files), bzl (for .bzl files), workspace (for WORKSPACE files), module (for MODULE.bazel files), default (for generic Starlark files) or auto (default, based on the filename)")
	flags.StringVar(&c.ConfigPath, "config", "", "path to .buildifier.json config file")
	flags.Var(&c.AllowSort, "allowsort", "additional sort contexts to treat as safe")
	flags.Var(&c.DisableRewrites, "buildifier_disable", "list of buildifier rewrites to disable")

	return flags
}

// Validate checks that the input type, format, and lint modes are correctly
// set.  It computes the final set of warnings used for linting.  The tables
// package is configured as a side-effect.
func (c *Config) Validate(args []string) error {
	if err := ValidateInputType(&c.InputType); err != nil {
		return err
	}

	if err := ValidateFormat(&c.Format, &c.Mode); err != nil {
		return err
	}

	if err := ValidateModes(&c.Mode, &c.Lint, &c.DiffMode); err != nil {
		return err
	}

	// If the path flag is set, must only be formatting a single file.
	// It doesn't make sense for multiple files to have the same path.
	if (c.WorkspaceRelativePath != "" || c.Mode == "print_if_changed") && len(args) > 1 {
		return fmt.Errorf("can only format one file when using -path flag or -mode=print_if_changed")
	}

	if c.TablesPath != "" {
		foundTablesPath, err := findTablesPath(c.TablesPath)
		if err != nil {
			return fmt.Errorf("failed to find %s for -tables: %w", c.TablesPath, err)
		}
		if err := tables.ParseAndUpdateJSONDefinitions(foundTablesPath, false); err != nil {
			return fmt.Errorf("failed to parse %s for -tables: %w", foundTablesPath, err)
		}
	}

	if c.AddTablesPath != "" {
		foundTablesPath, err := findTablesPath(c.AddTablesPath)
		if err != nil {
			return fmt.Errorf("failed to find %s for -add_tables: %w", c.AddTablesPath, err)
		}
		if err := tables.ParseAndUpdateJSONDefinitions(foundTablesPath, true); err != nil {
			return fmt.Errorf("failed to parse %s for -add_tables: %w", foundTablesPath, err)
		}
	}

	warningsList := c.WarningsList
	if c.Warnings != "" {
		warningsList = append(warningsList, c.Warnings)
	}
	warnings := strings.Join(warningsList, ",")
	lintWarnings, err := ValidateWarnings(&warnings, &warn.AllWarnings, &warn.DefaultWarnings)
	if err != nil {
		return err // TODO(pcj) return nil?
	}
	c.LintWarnings = lintWarnings

	return nil
}

// String renders the config as a formatted JSON string and satisfies the
// Stringer interface.
func (c *Config) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		log.Panicf("config marshal json: %v", err)
	}
	return string(data)
}

// ArrayFlags is a string slice that satisfies the flag.Value interface
type ArrayFlags []string

// String implements part of the flag.Value interface
func (i *ArrayFlags) String() string {
	return strings.Join(*i, ",")
}

// Set implements part of the flag.Value interface
func (i *ArrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// Example creates a sample configuration file for the -config=example flag.
func Example() *Config {
	c := New()
	c.InputType = "auto"
	c.Mode = "fix"
	c.Lint = "fix"
	c.WarningsList = warn.AllWarnings
	return c
}
