#!/bin/bash

set -u

go build
if [ $? -ne 0 ]; then
    echo "can't build project"
    exit 1
fi

TESTS=$(mktemp -d --suffix=".gunter.tests")

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
GUNTER_LOG=$(./gunter -c $TESTS/config -t $TESTS/templates -r 2>&1)
if [ $? -ne 0 ]; then
    echo "gunter running failed"
    echo "$GUNTER_LOG"
    exit 1
fi

GUNTER_TEMP_DIR=$(awk '{print $8}' <<< "$GUNTER_LOG")

permissions() {
    DIR=$1
    ls -lR $DIR | sed "s@$DIR@@" | awk '{print $1, $2, $3, $4}'
}

PERMISSIONS_EXPECTED=$(permissions $TESTS/templates/)
PERMISSIONS_ACTUAL=$(permissions $GUNTER_TEMP_DIR)

diff -u <(echo "$PERMISSIONS_EXPECTED") <(echo "$PERMISSIONS_ACTUAL")
if [ $? -ne 0 ]; then
    echo "permissions and ownership copy error"
    exit 1
fi

diff -u $TESTS/expected_bar_template $GUNTER_TEMP_DIR/dirbar/bar
if [ $? -ne 0 ]; then
    echo "template file compilation corrupted"
    exit 1
fi

# check that '.git.template' copied as '.git'
test -d $GUNTER_TEMP_DIR/.git
if [ $? -ne 0 ]; then
    echo ".git directory not copied"
    exit 1
fi

test ! -d $GUNTER_TEMP_DIR/.git.template
if [ $? -ne 0 ]; then
    echo ".git.template directory copied with suffix .template"
    exit 1
fi

# check backup
DEST_DIR=$(mktemp -d --suffix=".gunter.dest")
mkdir $DEST_DIR/dirbar
echo -n 'backup me' > $DEST_DIR/dirbar/bar

BACKUP_DIR=$(mktemp -d --suffix=".gunter.backup")

GUNTER_LOG=$(
    ./gunter \
        -c $TESTS/config -t $TESTS/templates \
        -d $DEST_DIR -b $BACKUP_DIR 2>&1
)
if [ $? -ne 0 ]; then
    echo "gunter running failed"
    echo "$GUNTER_LOG"
    exit 1
fi

test -f $BACKUP_DIR/dirbar/bar
if [ $? -ne 0 ]; then
    echo "dirbar/bar file not backuped"
    exit 1
fi

test ! -d $BACKUP_DIR/dirfoo
if [ $? -ne 0 ]; then
    echo "dirfoo directory should not be copied"
    exit 1
fi

test ! -d $BACKUP_DIR/.git
if [ $? -ne 0 ]; then
    echo ".git directory should not be copied"
    exit 1
fi

diff -u <(echo -n "backup me") $BACKUP_DIR/dirbar/bar
if [ $? -ne 0 ]; then
    echo "backuping failed"
    exit 1
fi

rm -rf $BACKUP_DIR $DEST_DIR
mkdir -p $DEST_DIR

GUNTER_LOG=$(
    ./gunter \
        -c $TESTS/config -t $TESTS/broken_templates \
        -d $DEST_DIR -b $BACKUP_DIR 2>&1
)
if [ $? -eq 0 ]; then
    echo "gunter doesn't fail when works with broken templates"
    exit 1
fi

grep -q "UnknownConfigField" <<< "$GUNTER_LOG"
if [ $? -ne 0 ]; then
    echo "gunter stderr doesn't contains message about UnknownConfigField"
    exit 1
fi

rm -rf $BACKUP_DIR $DEST_DIR

echo "tests passed"
