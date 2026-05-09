#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_dir="$(cd "${script_dir}/.." && pwd)"
source_svg="${project_dir}/build/app-icon.svg"
build_dir="${project_dir}/build"
iconset_dir="${build_dir}/icon.iconset"
tmp_dir="$(mktemp -d)"

cleanup() {
  rm -rf "${tmp_dir}"
}
trap cleanup EXIT

mkdir -p "${build_dir}"

if [[ ! -f "${source_svg}" ]]; then
  echo "missing icon source: ${source_svg}" >&2
  exit 1
fi

qlmanage -t -s 1024 -o "${tmp_dir}" "${source_svg}" >/dev/null 2>&1
rendered_png="${tmp_dir}/$(basename "${source_svg}").png"

if [[ ! -f "${rendered_png}" ]]; then
  echo "failed to render ${source_svg} with qlmanage" >&2
  exit 1
fi

rm -rf "${iconset_dir}"
mkdir -p "${iconset_dir}"
cp "${rendered_png}" "${build_dir}/app-icon.png"

sips -z 16 16 "${rendered_png}" --out "${iconset_dir}/icon_16x16.png" >/dev/null
sips -z 32 32 "${rendered_png}" --out "${iconset_dir}/icon_16x16@2x.png" >/dev/null
sips -z 32 32 "${rendered_png}" --out "${iconset_dir}/icon_32x32.png" >/dev/null
sips -z 64 64 "${rendered_png}" --out "${iconset_dir}/icon_32x32@2x.png" >/dev/null
sips -z 128 128 "${rendered_png}" --out "${iconset_dir}/icon_128x128.png" >/dev/null
sips -z 256 256 "${rendered_png}" --out "${iconset_dir}/icon_128x128@2x.png" >/dev/null
sips -z 256 256 "${rendered_png}" --out "${iconset_dir}/icon_256x256.png" >/dev/null
sips -z 512 512 "${rendered_png}" --out "${iconset_dir}/icon_256x256@2x.png" >/dev/null
sips -z 512 512 "${rendered_png}" --out "${iconset_dir}/icon_512x512.png" >/dev/null
sips -z 1024 1024 "${rendered_png}" --out "${iconset_dir}/icon_512x512@2x.png" >/dev/null

iconutil -c icns "${iconset_dir}" -o "${build_dir}/icon.icns"
rm -rf "${iconset_dir}"

echo "Generated ${build_dir}/icon.icns"
