#!/bin/bash

set -e -u

go build

TESTS=$(mktemp -d)

cp -r tests/* $TESTS

# gunter should copy empty directories too, but git cannot stage empty
# directories without any files, so .gitignore should be removed from template
# source directory.
rm $TESTS/templates/dirfoo/.gitignore

# gunter should copy file permissions and ownership
chown daemon:daemon $TESTS/templates/dirfoo
chmod u+s $TESTS/templates/dirfoo

chown $(id -u):daemon $TESTS/templates/dirbar/bar.template
chmod +x $TESTS/templates/dirbar/bar.template

chown $(id -u):daemon $TESTS/templates/.git.template/file_in_dot_git
chmod +x $TESTS/templates/.git.template/file_in_dot_git

# running gunter in dry run mode
DRYRUN=$(./gunter -c $TESTS/config -t $TESTS/templates -r 2>&1)
if [ $? -ne 0 ]; then
    echo "gunter running failed"
    echo "$DRYRUN"
    exit 1
fi

GUNTER_TEMP_DIR=$(awk '{print $10}' <<< "$DRYRUN")

permissions() {
    DIR=$1
    ls -lR $DIR | sed "s@$DIR@@" | awk '{print $1, $2, $3, $4}'
}

PERMISSIONS_EXPECTED=$(permissions $TESTS/templates/)
PERMISSIONS_ACTUAL=$(permissions $GUNTER_TEMP_DIR)

# -e flag should be restored because diff exits with status 1 if files are
# different.
set +e

diff -u <(echo "$PERMISSIONS_EXPECTED") <(echo "$PERMISSIONS_ACTUAL")
if [ $? -eq 1 ]; then
    echo "permissions and ownership copy error"
    exit 1
fi

diff -u $TESTS/expected_template $GUNTER_TEMP_DIR/dirbar/bar
if [ $? -eq 1 ]; then
    echo "template file compilation corrupted"
    exit 1
fi

set -e

# check that '.git.template' copied as '.git'
test -d $GUNTER_TEMP_DIR/.git

echo "tests passed"
