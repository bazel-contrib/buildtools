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

package bzlmod

import (
	"reflect"
	"strconv"
	"testing"

	"github.com/bazel-contrib/buildtools/v10/build"
)

const proxiesNoModuleHeader = ``

const proxiesModuleNameHeader = `
module(name = "name")
`

const proxiesModuleRepoNameHeader = `
module(name = "name", repo_name = "repo_name")
`

const proxiesBody = `
prox1 = use_extension("@name//bzl:extensions.bzl", "ext")
prox1.use(name = "foo")
isolated_prox1 = use_extension("@name//bzl:extensions.bzl", "ext", dev_dependency = False, isolate = True)
isolated_prox1.use(name = "foo")
prox2 = use_extension("@name//bzl:extensions.bzl", "ext", dev_dependency = True)
prox2.use(list = ["foo", "bar"])
# some comment
prox3 = use_extension("@repo_name//bzl:extensions.bzl", "ext")
prox3.use(label = "@dep//:bar")
isolated_prox2 = use_extension("@name//bzl:extensions.bzl", "ext", dev_dependency = True, isolate = True)
isolated_prox2.use(name = "foo")
prox4 = use_extension("@repo_name//bzl:extensions.bzl", "ext", dev_dependency = True)
prox4.use(dict = {"foo": "bar"})
isolated_prox3 = use_extension("@name//bzl:extensions.bzl", "ext", dev_dependency = True, isolate = True)
isolated_prox3.use(name = "foo")
prox5 = use_extension("@//bzl:extensions.bzl", "ext")
prox5.use(name = "foo")
prox6 = use_extension("@//bzl:extensions.bzl", "ext", dev_dependency = True)
prox6.use(list = ["foo", "bar"])
prox7 = use_extension("//bzl:extensions.bzl", "ext", dev_dependency = False)
prox7.use(label = "@foo//:bar")
isolated_prox4 = use_extension("@name//bzl:extensions.bzl", "ext", isolate = True)
isolated_prox4.use(name = "foo")
prox8 = use_extension("//bzl:extensions.bzl", "ext", dev_dependency = True)
unused = prox8.use(dict = {"foo": "bar"})
prox9 = use_extension(
    # comment
    "@dep//:extensions.bzl", "other_ext")
prox9.use(label = "@name//:bar")
prox10 = use_extension("@dep//:extensions.bzl", "other_ext", dev_dependency = bool(1))
prox10.use(dict = {"foo": "bar"})
prox11 = use_extension("extension.bzl", "ext")
prox12 = use_extension(":extension.bzl", "ext")
prox13 = use_extension("//:extension.bzl", "ext")
prox14 = use_extension("@name//:extension.bzl", "ext")
prox15 = use_extension("@repo_name//:extension.bzl", "ext")
`

func TestProxies(t *testing.T) {
	for i, tc := range []struct {
		content         string
		extBzlFiles     []string
		extName         string
		dev             bool
		expectedProxies []string
	}{
		{
			proxiesNoModuleHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl"},
			"ext",
			false,
			[]string{"prox5", "prox7"},
		},
		{
			proxiesNoModuleHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl"},
			"ext",
			true,
			[]string{"prox6", "prox8"},
		},
		{
			proxiesModuleNameHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl", "@name//bzl:extensions.bzl"},
			"ext",
			false,
			[]string{"prox1", "prox5", "prox7"},
		},
		{
			proxiesModuleNameHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl", "@name//bzl:extensions.bzl"},
			"ext",
			true,
			[]string{"prox2", "prox6", "prox8"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl", "@repo_name//bzl:extensions.bzl"},
			"ext",
			false,
			[]string{"prox3", "prox5", "prox7"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"//bzl:extensions.bzl", "@//bzl:extensions.bzl", "@repo_name//bzl:extensions.bzl"},
			"ext",
			true,
			[]string{"prox4", "prox6", "prox8"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"@name//bzl:extensions.bzl"},
			"ext",
			false,
			[]string{"prox1"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"@dep//:extensions.bzl"},
			"other_ext",
			false,
			[]string{"prox9"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"@dep//:extensions.bzl"},
			"other_ext",
			true,
			[]string{"prox10"},
		},
		{
			proxiesModuleNameHeader + proxiesBody,
			[]string{"//:extension.bzl", "@//:extension.bzl"},
			"ext",
			false,
			[]string{"prox11", "prox12", "prox13", "prox14"},
		},
		{
			proxiesModuleRepoNameHeader + proxiesBody,
			[]string{"//:extension.bzl", "@//:extension.bzl"},
			"ext",
			false,
			[]string{"prox11", "prox12", "prox13", "prox15"},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			for _, extBzlFile := range tc.extBzlFiles {
				t.Run("label_"+extBzlFile, func(t *testing.T) {

					f, err := build.ParseModule("MODULE.bazel", []byte(tc.content))
					if err != nil {
						t.Fatal(err)
					}
					actualProxies := Proxies(f, extBzlFile, tc.extName, tc.dev)
					if !reflect.DeepEqual(actualProxies, tc.expectedProxies) {
						t.Error("want: ", tc.expectedProxies, ", got: ", actualProxies)
					}
				})
			}
		})
	}
}

func TestAllProxies(t *testing.T) {
	for i, tc := range []struct {
		proxy           string
		expectedProxies []string
	}{
		{
			"invalid_proxy",
			nil,
		},
		{
			"isolated_prox1",
			[]string{"isolated_prox1"},
		},
		{
			"isolated_prox2",
			[]string{"isolated_prox2"},
		},
		{
			"isolated_prox3",
			[]string{"isolated_prox3"},
		},
		{
			"isolated_prox4",
			[]string{"isolated_prox4"},
		},
		{
			"prox1",
			[]string{"prox1", "prox5", "prox7"},
		},
		{
			"prox2",
			[]string{"prox2", "prox6", "prox8"},
		},
		{
			"prox9",
			[]string{"prox9"},
		},
		{
			"prox10",
			[]string{"prox10"},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := build.ParseModule("MODULE.bazel", []byte(proxiesModuleNameHeader+proxiesBody))
			if err != nil {
				t.Fatal(err)
			}
			proxies := AllProxies(f, tc.proxy)
			if !reflect.DeepEqual(proxies, tc.expectedProxies) {
				t.Error("want: ", tc.expectedProxies, ", got: ", proxies)
			}
		})
	}
}

const useReposFile = `
prox1 = use_extension("@mod//bzl:extensions.bzl", "ext")
prox1.use()
prox1.config()
prox2 = use_extension("//bzl:extensions.bzl", "ext")
use_repo(prox2)
use_repo(prox1, "repo5")
prox3 = use_extension("@dep//bzl:extensions.bzl", "ext")
prox2_dev = use_extension("//bzl:extensions.bzl", "ext", dev_dependency = True)
use_repo(prox1, "repo1")
use_repo(prox3, "repo2")
use_repo(prox2, "repo3", "repo4")
`

func TestUseRepos(t *testing.T) {
	for i, tc := range []struct {
		content       string
		proxies       []string
		expectedStmts []int
	}{
		{
			useReposFile,
			[]string{"prox2"},
			[]int{4, 10},
		},
		{
			useReposFile,
			[]string{"prox1", "prox3"},
			[]int{5, 8, 9},
		},
		{
			useReposFile,
			[]string{"prox2_dev"},
			[]int{},
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := build.ParseModule("MODULE.bazel", []byte(tc.content))
			if err != nil {
				t.Fatal(err)
			}
			var expectedUseRepos []*build.CallExpr
			for _, stmt := range tc.expectedStmts {
				expectedUseRepos = append(expectedUseRepos, f.Stmt[stmt].(*build.CallExpr))
			}
			actualUseRepos := UseRepos(f, tc.proxies)
			if !reflect.DeepEqual(actualUseRepos, expectedUseRepos) {
				t.Error("want: ", expectedUseRepos, ", got: ", actualUseRepos)
			}
		})
	}
}

func TestNewUseRepos(t *testing.T) {
	for i, tc := range []struct {
		content         string
		proxies         []string
		expectedContent string
		expectedUseRepo int
	}{
		{
			``,
			[]string{"prox1"},
			``,
			-1,
		},
		{
			`prox1 = use_extension("@module//:lib.bzl", "ext")`,
			[]string{"prox2"},
			`prox1 = use_extension("@module//:lib.bzl", "ext")
`,
			-1,
		},
		{
			`prox1 = use_extension("@mod//bzl:extensions.bzl", "ext", dev_dependency = True)`,
			[]string{"prox1"},
			`prox1 = use_extension("@mod//bzl:extensions.bzl", "ext", dev_dependency = True)
use_repo(prox1)
`,
			1,
		},
		{
			`prox1 = use_extension("@mod//bzl:extensions.bzl", "ext")
prox1.config()
prox1.download(name = "foo")`,
			[]string{"prox1"},
			`prox1 = use_extension("@mod//bzl:extensions.bzl", "ext")
prox1.config()
prox1.download(name = "foo")
use_repo(prox1)
`,
			3,
		},
		{
			`go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")

pull = use_extension("@rules_oci//oci:pull.bzl", "go_deps")
pull.oci_pull(name = "distroless_base")
`,
			[]string{"go_deps"},
			`go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")
use_repo(go_deps)

pull = use_extension("@rules_oci//oci:pull.bzl", "go_deps")
pull.oci_pull(name = "distroless_base")
`,
			2,
		},
		{
			`go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")

pull = use_extension("@rules_oci//oci:pull.bzl", "go_deps")
`,
			[]string{"go_deps"},
			`go_deps = use_extension("@gazelle//:extensions.bzl", "go_deps")
go_deps.from_file(go_mod = "//:go.mod")
use_repo(go_deps)

pull = use_extension("@rules_oci//oci:pull.bzl", "go_deps")
`,
			2,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := build.ParseModule("MODULE.bazel", []byte(tc.content))
			if err != nil {
				t.Fatal(err)
			}
			f, actualNewUseRepo := NewUseRepo(f, tc.proxies)
			actualContent := string(build.Format(f))
			if actualNewUseRepo != nil {
				if !reflect.DeepEqual(actualNewUseRepo, f.Stmt[tc.expectedUseRepo]) {
					t.Error("want: ", f.Stmt[tc.expectedUseRepo], ", got: ", actualNewUseRepo)
				}
			} else {
				if tc.expectedUseRepo != -1 {
					t.Error("wanted a nil new use_repo")
				}
			}
			if !reflect.DeepEqual(actualContent, tc.expectedContent) {
				t.Errorf("want:\n%q\ngot:\n%q\n", tc.expectedContent, actualContent)
			}
		})
	}
}

func TestAddRepoUsages(t *testing.T) {
	for i, tc := range []struct {
		content         string
		repos           []string
		expectedContent string
	}{
		{
			``,
			[]string{},
			``,
		},
		{
			`use_repo(prox)`,
			[]string{"repo2", "repo1"},
			`use_repo(prox, "repo1", "repo2")
`,
		},
		{
			`use_repo(
    prox,
)`,
			[]string{"repo2", "repo1"},
			`use_repo(
    prox,
    "repo1",
    "repo2",
)
`,
		},
		{
			`use_repo(prox, "repo2")`,
			[]string{"repo2", "repo1"},
			`use_repo(prox, "repo1", "repo2")
`,
		},
		{
			`use_repo(
    prox,
    "repo2",
)`,
			[]string{"repo2", "repo1"},
			`use_repo(
    prox,
    "repo1",
    "repo2",
)
`,
		},
		{
			`use_repo(prox, "repo1")
use_repo(
    prox2,
    my_repo = "repo2",
    "repo5",
)
use_repo(prox, "repo3")`,
			[]string{"repo3", "repo2", "repo4"},
			`use_repo(prox, "repo1")

use_repo(
    prox2,
    "repo5",
    my_repo = "repo2",
)

use_repo(prox, "repo3", "repo4")
`,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := build.ParseModule("MODULE.bazel", []byte(tc.content))
			if err != nil {
				t.Fatal(err)
			}
			var useRepos []*build.CallExpr
			for _, stmt := range f.Stmt {
				useRepos = append(useRepos, stmt.(*build.CallExpr))
			}
			AddRepoUsages(useRepos, tc.repos...)
			actualContent := string(build.Format(f))
			if !reflect.DeepEqual(actualContent, tc.expectedContent) {
				t.Errorf("want:\n%q\ngot:\n%q\n", tc.expectedContent, actualContent)
			}
		})
	}
}

func TestRemoveRepoUsages(t *testing.T) {
	for i, tc := range []struct {
		content         string
		repos           []string
		expectedContent string
	}{
		{
			``,
			[]string{"repo1"},
			``,
		},
		{
			`use_repo(prox)`,
			[]string{"repo2", "repo1"},
			`use_repo(prox)
`,
		},
		{
			`use_repo(
    prox,
)`,
			[]string{"repo2", "repo1"},
			`use_repo(
    prox,
)
`,
		},
		{
			`use_repo(prox, "repo2")`,
			[]string{"repo2", "repo1"},
			`use_repo(prox)
`,
		},
		{
			`use_repo(
    prox,
    "repo2",
)`,
			[]string{"repo2", "repo1"},
			`use_repo(prox)
`,
		},
		{
			`use_repo(prox, "repo1")
use_repo(
    prox2,
    my_repo = "repo2",
    "repo5",
)
use_repo(prox, "repo3")`,
			[]string{"repo3", "repo2", "repo4"},
			`use_repo(prox, "repo1")

use_repo(
    prox2,
    "repo5",
)

use_repo(prox)
`,
		},
	} {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			f, err := build.ParseModule("MODULE.bazel", []byte(tc.content))
			if err != nil {
				t.Fatal(err)
			}
			var useRepos []*build.CallExpr
			for _, stmt := range f.Stmt {
				useRepos = append(useRepos, stmt.(*build.CallExpr))
			}
			RemoveRepoUsages(useRepos, tc.repos...)
			actualContent := string(build.Format(f))
			if !reflect.DeepEqual(actualContent, tc.expectedContent) {
				t.Errorf("want:\n%q\ngot:\n%q\n", tc.expectedContent, actualContent)
			}
		})
	}
}
