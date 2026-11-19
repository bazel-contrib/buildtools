#!/usr/bin/env bash

if [[ -z "${BUILD_WORKING_DIRECTORY+x}" ]]; then
  echo "BUILD_WORKING_DIRECTORY is not set. Please run this target with bazel run."
  exit 1
fi

# Disable MSYS path conversion, which can otherwise modify Bazel labels.
export MSYS2_ARG_CONV_EXCL='*'
export MSYS_NO_PATHCONV=1

# --- begin runfiles.bash initialization v3 ---
set -uo pipefail; set +e; f=bazel_tools/tools/bash/runfiles/runfiles.bash
# shellcheck disable=SC1090
source "${RUNFILES_DIR:-/dev/null}/$f" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "${RUNFILES_MANIFEST_FILE:-/dev/null}" | cut -f2- -d' ')" 2>/dev/null || \
  source "$0.runfiles/$f" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "$0.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  source "$(grep -sm1 "^$f " "$0.exe.runfiles_manifest" | cut -f2- -d' ')" 2>/dev/null || \
  { echo >&2 "ERROR: cannot find $f"; exit 1; }; f=; set -e
# --- end runfiles.bash initialization v3 ---

buildozer_rlocationpath=buildozer_binary/buildozer.exe
buildozer_path="$(rlocation "$buildozer_rlocationpath")"
if [[ ! -f "$buildozer_path" ]]; then
  echo "buildozer.exe not found at runfiles path $buildozer_rlocationpath and resolved path $buildozer_path."
  exit 1
fi

cd "$BUILD_WORKING_DIRECTORY"
exec "$buildozer_path" "$@"
