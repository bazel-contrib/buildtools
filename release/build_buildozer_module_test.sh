#!/usr/bin/env bash

set -o errexit -o nounset -o pipefail

runfile() {
  echo "${TEST_SRCDIR}/${TEST_WORKSPACE}/$1"
}

PACKAGER="$(runfile "${PACKAGER}")"
readonly PACKAGER
BUILDOZER="$(runfile "${BUILDOZER}")"
readonly BUILDOZER
MODULE_FILE="$(runfile "${MODULE_FILE}")"
readonly MODULE_FILE
TEST_ROOT="$(mktemp -d "${TEST_TMPDIR}/module-package.XXXXXX")"
readonly TEST_ROOT
readonly MODULE_SOURCE="${TEST_ROOT}/module"
readonly DIST="${TEST_ROOT}/dist"
readonly VERSION=99.1.2
readonly PREFIX="buildozer-${VERSION}"

mkdir -p "${MODULE_SOURCE}" "${DIST}"
cp -RL "$(dirname "${MODULE_FILE}")/." "${MODULE_SOURCE}"
touch "${MODULE_SOURCE}/MODULE.bazel.lock"
ln -s nowhere "${MODULE_SOURCE}/bazel-module"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) editor_platform=darwin-arm64 ;;
  Darwin-x86_64) editor_platform=darwin-amd64 ;;
  Linux-x86_64) editor_platform=linux-amd64 ;;
  Linux-aarch64) editor_platform=linux-arm64 ;;
  *) echo "Unsupported test host" >&2; exit 1 ;;
esac

platforms=(
  darwin-amd64
  darwin-arm64
  linux-amd64
  linux-arm64
  linux-riscv64
  linux-s390x
  windows-amd64
  windows-arm64
)
for platform in "${platforms[@]}"; do
  asset="buildozer-${platform}"
  if [[ "${platform}" == windows-* ]]; then
    asset="${asset}.exe"
  fi
  cp "${BUILDOZER}" "${DIST}/${asset}"
done
chmod +x "${DIST}/buildozer-${editor_platform}"

"${PACKAGER}" "${VERSION}" "${DIST}" "${MODULE_SOURCE}"

mkdir "${TEST_ROOT}/extracted"
tar -xzf "${DIST}/buildozer-v${VERSION}.tar.gz" -C "${TEST_ROOT}/extracted"
readonly EXTRACTED="${TEST_ROOT}/extracted/${PREFIX}"

if command -v sha256sum >/dev/null 2>&1; then
  digest="$(sha256sum "${BUILDOZER}" | cut -d' ' -f1)"
else
  digest="$(shasum -a 256 "${BUILDOZER}" | cut -d' ' -f1)"
fi

[[ "$(grep -c "${digest}" "${EXTRACTED}/MODULE.bazel")" -eq 8 ]]
grep -q 'version = "99.1.2"' "${EXTRACTED}/MODULE.bazel"
grep -q 'version = "99.1.2"' "${EXTRACTED}/README.md"
[[ -f "${EXTRACTED}/LICENSE" ]]
[[ ! -e "${EXTRACTED}/MODULE.bazel.lock" ]]
[[ ! -e "${EXTRACTED}/bazel-module" ]]
