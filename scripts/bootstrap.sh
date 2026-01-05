#!/usr/bin/env bash
set -euo pipefail

exe=""
case "$(go env GOOS)" in
  windows) exe=".exe" ;;
esac

go build -o "./tetra${exe}" ./cli/cmd/tetra
echo "Built: ./tetra${exe}"
