#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
project_dir="$(cd "${script_dir}/.." && pwd)"
source_png="${project_dir}/build/app-icon.png"
build_dir="${project_dir}/build"
public_dir="${project_dir}/public"
iconset_dir="${build_dir}/icon.iconset"

if [[ ! -f "${source_png}" ]]; then
  echo "missing icon source: ${source_png}" >&2
  exit 1
fi

width="$(sips -g pixelWidth "${source_png}" | awk '/pixelWidth/ {print $2}')"
height="$(sips -g pixelHeight "${source_png}" | awk '/pixelHeight/ {print $2}')"

if [[ "${width}" != "${height}" ]]; then
  echo "icon source must be square, got ${width}x${height}: ${source_png}" >&2
  exit 1
fi

rm -rf "${iconset_dir}"
mkdir -p "${iconset_dir}" "${public_dir}"

sips -z 16 16     "${source_png}" --out "${iconset_dir}/icon_16x16.png"      >/dev/null
sips -z 32 32     "${source_png}" --out "${iconset_dir}/icon_16x16@2x.png"   >/dev/null
sips -z 32 32     "${source_png}" --out "${iconset_dir}/icon_32x32.png"      >/dev/null
sips -z 64 64     "${source_png}" --out "${iconset_dir}/icon_32x32@2x.png"   >/dev/null
sips -z 128 128   "${source_png}" --out "${iconset_dir}/icon_128x128.png"    >/dev/null
sips -z 256 256   "${source_png}" --out "${iconset_dir}/icon_128x128@2x.png" >/dev/null
sips -z 256 256   "${source_png}" --out "${iconset_dir}/icon_256x256.png"    >/dev/null
sips -z 512 512   "${source_png}" --out "${iconset_dir}/icon_256x256@2x.png" >/dev/null
sips -z 512 512   "${source_png}" --out "${iconset_dir}/icon_512x512.png"    >/dev/null
sips -z 1024 1024 "${source_png}" --out "${iconset_dir}/icon_512x512@2x.png" >/dev/null

iconutil -c icns "${iconset_dir}" -o "${build_dir}/icon.icns"
rm -rf "${iconset_dir}"

sips -z 256 256 "${source_png}" --out "${public_dir}/favicon.png" >/dev/null

echo "Generated ${build_dir}/icon.icns"
echo "Generated ${public_dir}/favicon.png"
