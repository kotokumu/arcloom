#!/usr/bin/env bash
set -euo pipefail
repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)
cd "$repo_dir"
rustup target add wasm32-unknown-unknown
cargo build --locked -p arcloom-os
cargo build --locked --release --target wasm32-unknown-unknown -p arcloom-feedback-loop -p arcloom-observe-only
build_dir=${CARGO_TARGET_DIR:-target}
mkdir -p agent-os/examples/generated
"$build_dir/debug/arcloom-os" pack "$build_dir/wasm32-unknown-unknown/release/arcloom_feedback_loop.wasm" agent-os/examples/generated/feedback-loop.component.wasm
"$build_dir/debug/arcloom-os" pack "$build_dir/wasm32-unknown-unknown/release/arcloom_observe_only.wasm" agent-os/examples/generated/observe-only.component.wasm
run_dir=$(mktemp -d)
printf 'State directory: %s\n' "$run_dir"
"$build_dir/debug/arcloom-os" run agent-os/examples/feedback-loop.json "$run_dir" feedback-demo
"$build_dir/debug/arcloom-os" inspect "$run_dir" feedback-demo
"$build_dir/debug/arcloom-os" run agent-os/examples/observe-only.json "$run_dir" observe-demo
"$build_dir/debug/arcloom-os" inspect "$run_dir" observe-demo
