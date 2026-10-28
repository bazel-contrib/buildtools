#!/usr/bin/env bash

# Verify that the buildozer module has no unexpected non-dev dependencies.

set -euo pipefail

# Fail on lines that don't end with " True".
! "$BUILDOZER" 'print name dev_dependency' //MODULE.bazel:%bazel_dep \
  | grep -v 'bazel_features ' \
  | grep -v 'rules_shell ' \
  | grep -v ' True$'
