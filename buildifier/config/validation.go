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

package config

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// ValidateInputType validates the value of --type
func ValidateInputType(inputType *string) error {
	switch *inputType {
	case "build", "bzl", "workspace", "default", "module", "repo", "vendor", "auto":
		return nil

	default:
		return fmt.Errorf("unrecognized input type %s; valid types are build, bzl, workspace, default, module, repo, vendor, auto", *inputType)
	}
}

// ValidateFormat validates the value of --format
func ValidateFormat(format, mode *string) error {
	switch *format {
	case "":
		return nil

	case "text", "json":
		if *mode != "check" {
			return fmt.Errorf("cannot specify --format without --mode=check")
		}

	default:
		return fmt.Errorf("unrecognized format %s; valid types are text, json", *format)
	}
	return nil
}

// isRecognizedMode checks whether the given mode is one of the valid modes.
func isRecognizedMode(validModes []string, mode string) bool {
	for _, m := range validModes {
		if mode == m {
			return true
		}
	}
	return false
}

// ValidateModes validates flags --mode, --lint, and -d
func ValidateModes(mode, lint *string, dflag *bool, additionalModes ...string) error {
	if *dflag {
		if *mode != "" {
			return fmt.Errorf("cannot specify both -d and -mode flags")
		}
		*mode = "diff"
	}

	// Check mode.
	validModes := []string{"check", "diff", "fix", "print_if_changed"}
	validModes = append(validModes, additionalModes...)

	if *mode == "" {
		*mode = "fix"
	} else if !isRecognizedMode(validModes, *mode) {
		return fmt.Errorf("unrecognized mode %s; valid modes are %s", *mode, strings.Join(validModes, ", "))
	}

	// Check lint mode.
	switch *lint {
	case "":
		*lint = "off"

	case "off", "warn":
		// ok

	case "fix":
		if *mode != "fix" {
			return fmt.Errorf("--lint=fix is only compatible with --mode=fix")
		}

	default:
		return fmt.Errorf("unrecognized lint mode %s; valid modes are warn and fix", *lint)
	}

	return nil
}

// ValidateWarnings validates the value of the --warnings flag
func ValidateWarnings(warnings *string, allWarnings, defaultWarnings *[]string) ([]string, error) {

	// Check lint warnings
	warningsMap := make(map[string]bool)
	switch *warnings {
	case "", "default":
		for _, w := range *defaultWarnings {
			warningsMap[w] = true
		}
	case "all":
		for _, w := range *allWarnings {
			warningsMap[w] = true
		}
	default:
		// Either all or no warning categories should start with "+" or "-".
		// If all of them start with "+" or "-", the semantics is
		// "default set of warnings + something - something".
		plus := map[string]bool{}
		minus := map[string]bool{}
		for _, warning := range strings.Split(*warnings, ",") {
			if strings.HasPrefix(warning, "+") {
				plus[warning[1:]] = true
			} else if strings.HasPrefix(warning, "-") {
				minus[warning[1:]] = true
			} else {
				warningsMap[warning] = true
			}
		}
		if len(warningsMap) > 0 && (len(plus) > 0 || len(minus) > 0) {
			return []string{}, fmt.Errorf("warning categories with modifiers (\"+\" or \"-\") can't be mixed with raw warning categories")
		}
		if len(warningsMap) == 0 {
			for _, warning := range *defaultWarnings {
				if !minus[warning] {
					warningsMap[warning] = true
				}
			}
			for warning := range plus {
				warningsMap[warning] = true
			}
		}
	}
	ws := slices.Collect(maps.Keys(warningsMap))
	// Sorts the warnings to ensure behaviors are deterministic.
	slices.Sort(ws)
	return ws, nil
}
