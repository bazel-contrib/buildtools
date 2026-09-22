#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

cd "$BUILD_WORKSPACE_DIRECTORY"

# Workaround for https://github.com/bazelbuild/bazel/issues/17571
unset RUNFILES_DIR
unset RUNFILES_MANIFEST_FILE
unset RUNFILES_MANIFEST_ONLY

bazel run @buildozer -- 'dict_set env SHOULD_PASS:1' //bazel_run_test
