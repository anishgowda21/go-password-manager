#!/usr/bin/env bash

set -euo pipefail

script_dir="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
client_public="$(cd -- "$script_dir/../client" && pwd)/public"
goroot="$(go env GOROOT)"

if [[ -f "$goroot/lib/wasm/wasm_exec.js" ]]; then
  wasm_exec="$goroot/lib/wasm/wasm_exec.js"
elif [[ -f "$goroot/misc/wasm/wasm_exec.js" ]]; then
  wasm_exec="$goroot/misc/wasm/wasm_exec.js"
else
  printf 'could not find wasm_exec.js under GOROOT: %s\n' "$goroot" >&2
  exit 1
fi

mkdir -p "$client_public"

(
  cd -- "$script_dir"
  GOOS=js GOARCH=wasm go build -o "$client_public/crypto.wasm" .
)
cp "$wasm_exec" "$client_public/wasm_exec.js"

printf 'Generated %s/crypto.wasm and %s/wasm_exec.js\n' "$client_public" "$client_public"