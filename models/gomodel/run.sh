#!/usr/bin/env bash
set -xeuo pipefail

docker run --rm -it --name gomodel \
  -p 9999:9999 \
  -v "$PWD/config.yaml:/app/config/config.yaml:ro" \
  enterpilot/gomodel:local