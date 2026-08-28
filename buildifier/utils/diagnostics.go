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

package utils

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bazelbuild/buildtools/build"
	"github.com/bazelbuild/buildtools/warn"
)

// Diagnostics contains diagnostic information returned by formatter and linter
type Diagnostics struct {
	Success bool               `json:"success"` // overall success (whether all files are formatted properly and have no warnings)
	Files   []*FileDiagnostics `json:"files"`   // diagnostics per file
}

// Format formats a Diagnostics object either as plain text or as json
func (d *Diagnostics) Format(format string, verbose bool) string {
	switch format {
	case "text", "":
		var output strings.Builder
		for _, f := range d.Files {
			for _, w := range f.Warnings {
				formatString := "%s:%d: %s: %s (%s)\n"
				if !w.Actionable {
					formatString = "%s:%d: %s: %s [%s]\n"
				}
				output.WriteString(fmt.Sprintf(formatString,
					f.Filename,
					w.Start.Line,
					w.Category,
					w.Message,
					w.URL))
			}
			if !f.Formatted {
				output.WriteString(fmt.Sprintf("%s # reformat\n", f.Filename))
			}
		}
		return output.String()
	case "json":
		var result []byte
		if verbose {
			result, _ = json.MarshalIndent(*d, "", "    ")
		} else {
			result, _ = json.Marshal(*d)
		}
		return string(result) + "\n"
	}
	return ""
}

// FileDiagnostics contains diagnostics information for a file
type FileDiagnostics struct {
	Filename  string     `json:"filename"`
	Formatted bool       `json:"formatted"`
	Valid     bool       `json:"valid"`
	Warnings  []*warning `json:"warnings"`
}

type warning struct {
	Start       position     `json:"start"`
	End         position     `json:"end"`
	Category    string       `json:"category"`
	Actionable  bool         `json:"actionable"`
	AutoFixable bool         `json:"autoFixable"`
	Message     string       `json:"message"`
	URL         string       `json:"url"`
    Replacement *replacement `json:"replacement,omitempty"` // Optional replacement
}

type position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type replacement struct {
	Start   int    `json:"start"`   // Start offset in bytes
	End     int    `json:"end"`     // End offset in bytes, may be equal to Start for insertions
	Content string `json:"content"` // Replacement content
}

// NewDiagnostics returns a new Diagnostics object
func NewDiagnostics(fileDiagnostics ...*FileDiagnostics) *Diagnostics {
	diagnostics := &Diagnostics{
		Success: true,
		Files:   fileDiagnostics,
	}
	for _, file := range diagnostics.Files {
		if !file.Formatted || len(file.Warnings) > 0 {
			diagnostics.Success = false
			break
		}
	}
	return diagnostics
}

// NewFileDiagnostics returns a new FileDiagnostics object
func NewFileDiagnostics(filename string, warnings []*warn.Finding) *FileDiagnostics {
	fileDiagnostics := FileDiagnostics{
		Filename:  filename,
		Formatted: true,
		Valid:     true,
		Warnings:  []*warning{},
	}

	for _, w := range warnings {
		fileDiagnostics.Warnings = append(fileDiagnostics.Warnings, &warning{
			Start:       makePosition(w.Start),
			End:         makePosition(w.End),
			Category:    w.Category,
			Actionable:  w.Actionable,
			AutoFixable: w.AutoFixable,
			Message:     w.Message,
			URL:         w.URL,
			Replacement: makeReplacement(w.Replacement),
		})
	}

	return &fileDiagnostics
}

// InvalidFileDiagnostics returns a new FileDiagnostics object for an invalid file
func InvalidFileDiagnostics(filename string) *FileDiagnostics {
	fileDiagnostics := &FileDiagnostics{
		Filename:  filename,
		Formatted: false,
		Valid:     false,
		Warnings:  []*warning{},
	}
	if filename == "" {
		fileDiagnostics.Filename = "<stdin>"
	}
	return fileDiagnostics
}

func makePosition(p build.Position) position {
	return position{
		Line:   p.Line,
		Column: p.LineRune,
	}
}

func makeReplacement(r *warn.Replacement) *replacement {
	if r == nil {
		return nil
	}
	return &replacement{
		Start:   r.Start,
		End:     r.End,
		Content: r.Content,
	}
}
