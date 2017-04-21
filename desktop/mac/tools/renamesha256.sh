#!/bin/bash
set -eo pipefail
digest=`shasum -a 256 "$1"|cut -b-64`
mv "$1" "$1"-$digest
