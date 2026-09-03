#!/bin/sh
unset GITHUB_TOKEN GH_TOKEN
plan_codex_state_dir=/private/tmp/arcloom-codex-state.32vhNJ
exec env CODEX_HOME="$plan_codex_state_dir" \
  /private/tmp/arcloom-m2.7Hu4F2/codex-aarch64-apple-darwin \
  -c 'features.plugins=false' \
  -c 'features.apps=false' \
  -c 'apps._default.enabled=false' \
  -c 'web_search="disabled"' \
  "$@"
