/*
Copyright 2026 Google LLC

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

package warn

import "testing"

func TestDeprecatedModuleExtension(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/extensions.bzl": `
def _ext_v1_impl(ctx):
  """
  Some extension.

  Deprecated:
    please use my_ext_v4 instead.
  """
  pass

my_ext_v1 = module_extension(
    implementation = _ext_v1_impl,
)

def _ext_v2_impl(ctx):
  """
  Some extension.

  Deprecated:
    please use my_ext_v4 instead.
  """
  pass

my_ext_v2 = module_extension(
    implementation = _ext_v2_impl,
)

def _ext_v3_impl(ctx):
  pass

my_ext_v3 = module_extension(
    implementation = _ext_v3_impl,
    doc = "Deprecated: please use my_ext_v4 instead.",
)

def _ext_v4_impl(ctx):
  pass

my_ext_v4 = module_extension(
    implementation = _ext_v4_impl,
)
`,
	})()

	checkFindings(t, "deprecated-module-ext", `
my_ext_v1 = use_extension(":extensions.bzl", "my_ext_v1")
my_ext_v3 = use_extension(":extensions.bzl", "my_ext_v3")
my_ext_v4 = use_extension(":extensions.bzl", "my_ext_v4")
`,
		[]string{
			`1: The module extension "my_ext_v1" defined in "//test/package/extensions.bzl" is deprecated.`,
			`2: The module extension "my_ext_v3" defined in "//test/package/extensions.bzl" is deprecated.`,
		},
		scopeModule)
}

func TestDeprecatedTagClass(t *testing.T) {
	defer setUpFileReader(map[string]string{
		"test/package/extensions.bzl": `
def _ext_impl(ctx):
  pass

my_ext = module_extension(
    implementation = _ext_impl,
    tag_classes = {
        "tag_v1": tag_class(
            doc = "Deprecated: use tag_v3 instead.",
        ),
        "tag_v2": tag_class(
            doc = "Deprecated: use tag_v3 instead.",
        ),
        "tag_v3": tag_class(),
    },
)
`,
	})()

	checkFindings(t, "deprecated-module-ext-tag", `
my_ext = use_extension(":extensions.bzl", "my_ext")
my_ext.tag_v1(name = "v1")
my_ext.tag_v3(name = "v3")
`,
		[]string{
			`2: The tag class "tag_v1" of module extension "my_ext" defined in "//test/package/extensions.bzl" is deprecated.`,
		},
		scopeModule)
}
