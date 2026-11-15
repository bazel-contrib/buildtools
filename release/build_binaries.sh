#!/usr/bin/env bash
#
# Builds the release binaries of buildifier, buildozer and unused_deps for all
# supported platforms and copies them into the directory given as the first
# argument, using the names under which they are attached to GitHub releases
# (e.g. buildifier-linux-amd64 or unused-deps-windows-amd64.exe).
#
# Usage: release/build_binaries.sh <output directory>

set -o errexit -o nounset -o pipefail

if [[ $# -ne 1 ]]; then
  echo "Usage: $0 <output directory>" >&2
  exit 1
fi

OUT_DIR="$(mkdir -p "$1" && cd "$1" && pwd)"
cd "$(dirname "$0")/.."

TOOLS=(buildifier buildozer unused_deps)
PLATFORMS=(
  darwin-amd64
  darwin-arm64
  linux-amd64
  linux-arm64
  linux-riscv64
  linux-s390x
  windows-amd64
  windows-arm64
)

TARGETS=()
for tool in "${TOOLS[@]}"; do
  for platform in "${PLATFORMS[@]}"; do
    TARGETS+=("//${tool}:${tool}-${platform}")
  done
done

bazel build --config=release -- "${TARGETS[@]}"

bazel cquery --config=release --output=files -- "set(${TARGETS[*]})" | while IFS= read -r output; do
  # Bazel output names use underscores (unused_deps-linux_amd64), whereas
  # release assets use dashes throughout (unused-deps-linux-amd64).
  name="$(basename "${output}")"
  cp "${output}" "${OUT_DIR}/${name//_/-}"
done

for tool in "${TOOLS[@]}"; do
  for platform in "${PLATFORMS[@]}"; do
    asset="${tool//_/-}-${platform}"
    if [[ "${platform}" == windows-* ]]; then
      asset="${asset}.exe"
    fi
    if [[ ! -f "${OUT_DIR}/${asset}" ]]; then
      echo "Expected release asset ${asset} was not built" >&2
      exit 1
    fi
  done
done

ls -l "${OUT_DIR}"
