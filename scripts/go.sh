#!/bin/sh
set -eu
if [ -x .tools/go/bin/go ]; then GOTOOLCHAIN=go1.27.2 exec .tools/go/bin/go "$@"; fi
GOTOOLCHAIN=go1.27.2 exec go "$@"
