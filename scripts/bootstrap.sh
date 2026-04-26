#!/usr/bin/env bash
set -euo pipefail

exe=""
case "$(go env GOOS)" in
  windows) exe=".exe" ;;
esac

go build -o "./tetra${exe}" ./cli/cmd/tetra
cp "./tetra${exe}" "./t${exe}"
echo "Built: ./tetra${exe} ./t${exe}"
