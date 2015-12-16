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

# Fail if we have more than one of any index
# on the same day. YYYY-MM-DD.N
dups=$(ls 201?-*|cut -b-12|sort|uniq -c|grep -v '1 ' || true)
if test -n "$dups"
then
	printf >&2 'duplicate indexes:\n'
	printf >&2 '%s\n' "$dups"
	exit 1
fi
