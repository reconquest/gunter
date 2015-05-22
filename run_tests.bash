#!/bin/bash

set -e

go build

TESTS=$(mktemp -d)

cp -r tests/* $TESTS

# gunter should copy empty directories too, but git cannot stage empty
# directories without any files, so should remove gitignore file in temporary
# tests directory
rm $TESTS/templates/dirfoo/.gitignore

# gunter should copy file permissions and ownership
chown daemon:daemon $TESTS/templates/dirfoo
chmod u+s $TESTS/templates/dirfoo

chown $(id -u):daemon $TESTS/templates/dirbar/template
chmod +x $TESTS/templates/dirbar/template

# running gunter in dry run mode
DRYRUN=$(./gunter -c $TESTS/config -t $TESTS/templates -r 2>&1)
GUNTER_TEMP_DIR=$(awk '{print $10}' <<< "$DRYRUN")

permissions() {
    ls -lR $1 | sed "s@$1@@" | awk '{print $1, $2, $3, $4, $9}'
}

PERMISSIONS_EXPECTED=$(permissions $TESTS/templates/)
PERMISSIONS_ACTUAL=$(permissions $GUNTER_TEMP_DIR)

set +e

diff -u <(echo "$PERMISSIONS_EXPECTED") <(echo "$PERMISSIONS_ACTUAL")
if [ $? -eq 1 ]; then
    echo "permissions and ownership copy error"
    exit 1
fi

diff -u $TESTS/expected_template $GUNTER_TEMP_DIR/dirbar/template
if [ $? -eq 1 ]; then
    echo "template file compilation corrupted"
    exit 1
fi

echo "okay"