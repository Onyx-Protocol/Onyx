#!/bin/bash

set -eo pipefail
shopt -s extglob

ok=true
for f in *
do
	case $f in
	Readme|check-names.bash)
		;; # skip
	201[5-9]-[0-9][0-9]-[0-9][0-9].+([0-9]).+([a-z]).+([a-z0-9-]).@(sql|go))
		;; # ok
	*)
		ok=false
		echo >&2 bad name: $f
		;;
	esac
done
if test $ok != true
then exit 1
fi
