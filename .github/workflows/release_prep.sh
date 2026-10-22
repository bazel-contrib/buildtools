#!/usr/bin/env bash

# Builds all files uploaded to a GitHub release. This script is invoked by the
# bazel-contrib release workflow and writes release notes to stdout.

set -o errexit -o nounset -o pipefail

if [[ $# -ne 1 || "$1" != v* ]]; then
  echo "Usage: $0 v<version>" >&2
  exit 1
fi

readonly TAG="$1"
readonly VERSION="${TAG#v}"

release/build_binaries.sh dist
release/build_buildozer_module.sh "${VERSION}" dist

cat <<EOF
## Using the buildozer Bazel module

Add the following to your \`MODULE.bazel\` file:

\`\`\`starlark
bazel_dep(name = "buildozer", version = "${VERSION}", dev_dependency = True)
\`\`\`
EOF
