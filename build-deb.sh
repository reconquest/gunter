#!/bin/bash

set -euo pipefail

builddir=`dirname "$(readlink -f "${BASH_SOURCE[0]}")"`

. "$builddir/deb-from-pkgbuild/build.sh"

:build

# custom changes
#pkgroot=`:get-package-root`
#rm "$pkgroot/somedir/somefile"

# add dependencies
#:set-deb-dependencies "git-core (>=1:2.3.7)"

:package-deb
