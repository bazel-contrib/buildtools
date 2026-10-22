#!/usr/bin/env bash
#
# Packages the buildozer Bazel module using the buildozer binaries in a release
# output directory. The generated MODULE.bazel pins each binary by SHA-256.
#
# Usage: release/build_buildozer_module.sh <version> <output directory> [module source]

set -o errexit -o nounset -o pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "Usage: $0 <version> <output directory> [module source]" >&2
  exit 1
fi

readonly VERSION="$1"
REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
readonly REPO_ROOT
OUT_DIR="$(mkdir -p "$2" && cd "$2" && pwd)"
readonly OUT_DIR
readonly MODULE_SOURCE="${3:-${REPO_ROOT}/buildozer/module}"
readonly PREFIX="buildozer-${VERSION}"
readonly ARCHIVE="${OUT_DIR}/buildozer-v${VERSION}.tar.gz"

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) editor_platform=darwin-arm64 ;;
  Darwin-x86_64) editor_platform=darwin-amd64 ;;
  Linux-x86_64) editor_platform=linux-amd64 ;;
  Linux-aarch64) editor_platform=linux-arm64 ;;
  *)
    echo "Unsupported release host: $(uname -s)-$(uname -m)" >&2
    exit 1
    ;;
esac

if command -v sha256sum >/dev/null 2>&1; then
  sha256() {
    sha256sum "$1" | cut -d' ' -f1
  }
else
  sha256() {
    shasum -a 256 "$1" | cut -d' ' -f1
  }
fi

STAGING_DIR="$(mktemp -d)"
readonly STAGING_DIR
COMMANDS_FILE="$(mktemp)"
readonly COMMANDS_FILE
trap 'rm -rf -- "${STAGING_DIR}"; rm -f -- "${COMMANDS_FILE}"' EXIT

readonly MODULE_DIR="${STAGING_DIR}/${PREFIX}"
mkdir -p "${MODULE_DIR}"
cp -R "${MODULE_SOURCE}/." "${MODULE_DIR}"
cp "${REPO_ROOT}/LICENSE" "${MODULE_DIR}/LICENSE"
# Do not package files created by local Bazel invocations.
find "${MODULE_DIR}" -name MODULE.bazel.lock -delete
find "${MODULE_DIR}" -type l -name 'bazel-*' -delete

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

sha256_dict=""
for platform in "${platforms[@]}"; do
  asset="buildozer-${platform}"
  if [[ "${platform}" == windows-* ]]; then
    asset="${asset}.exe"
  fi
  if [[ ! -f "${OUT_DIR}/${asset}" ]]; then
    echo "Expected release asset ${asset} was not found in ${OUT_DIR}" >&2
    exit 1
  fi
  digest="$(sha256 "${OUT_DIR}/${asset}")"
  sha256_dict="${sha256_dict:+${sha256_dict} }${platform}:${digest}"
done

cat >"${COMMANDS_FILE}" <<EOF
set version ${VERSION}|//MODULE.bazel:%buildozer_binary.buildozer
dict_set sha256 ${sha256_dict}|//MODULE.bazel:%buildozer_binary.buildozer
EOF

readonly EDITOR="${OUT_DIR}/buildozer-${editor_platform}"
if [[ ! -x "${EDITOR}" ]]; then
  echo "Buildozer editor ${EDITOR} is missing or not executable" >&2
  exit 1
fi

(
  cd "${MODULE_DIR}"
  "${EDITOR}" -f "${COMMANDS_FILE}"
)

sed -i.bak 's/version = "[^"]*"/version = "'"${VERSION}"'"/g' "${MODULE_DIR}/README.md"
rm "${MODULE_DIR}/README.md.bak"

tar -C "${STAGING_DIR}" -cf - "${PREFIX}" | gzip -n >"${ARCHIVE}"
echo "Created ${ARCHIVE}"
