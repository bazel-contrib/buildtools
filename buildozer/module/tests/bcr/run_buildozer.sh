#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

cd "$BUILD_WORKSPACE_DIRECTORY"

bazel run @buildozer -- 'dict_set env SHOULD_PASS:1' //bazel_run_test
