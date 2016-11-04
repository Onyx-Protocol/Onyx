#!/bin/sh

set -eu

fpm -n cored -v $VERSION \
  -s dir -t rpm \
  -p /output \
  --before-install /before-install \
  --after-install /after-install \
  --after-remove /after-remove \
  /usr/bin/cored /usr/bin/corectl /etc/init.d/cored
