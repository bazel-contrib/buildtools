// Package buildifier provides a Buildifier which doesn't call os.exec.
//
// There are some programming frameworks which consider os.exec dangerous.
//
// The package is imported just for its side-effects. For example:
//
// import _ "github.com/bazel-contrib/buildtools/v10/edit/safe/buildifier"
package buildifier

import (
	"github.com/bazel-contrib/buildtools/v10/build"
	"github.com/bazel-contrib/buildtools/v10/edit"
)

func init() {
	edit.RegisterBuildifier(&buildifier{})
}

type buildifier struct{}

// runBuildifier formats the build file f using the built-in formatter.
func (b *buildifier) Buildify(_ *edit.Options, f *build.File) ([]byte, error) {
	// Current AST may be not entirely correct, e.g. it may contain Ident which
	// value is a chunk of code, like "f(x)". The AST should be printed and
	// re-read to parse such expressions correctly.
	contents := build.Format(f)
	newF, err := build.Parse(f.Path, contents)
	if err != nil {
		return nil, err
	}
	return build.Format(newF), nil
}
