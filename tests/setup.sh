#!/bin/bash

_user=$(id --user --name)
_group=$(id --group --name)

tests:clone ../gunter.test bin

tests:put bin/chown <<MOCK
#!/bin/bash
echo "\$@" >> $(tests:get-tmp-dir)/chown.log
MOCK

tests:put bin/chmod <<MOCK
#!/bin/bash
echo "\$@" >> $(tests:get-tmp-dir)/chmod.log
MOCK

tests:ensure chmod +x bin/chmod bin/chown

tests:make-tmp-dir templates
tests:make-tmp-dir target
tests:make-tmp-dir backup

:gunter() {
    tests:ensure fakeroot gunter.test "$@"
    cat $(tests:get-stdout-file)
    cat $(tests:get-stderr-file) >&2
    return $(tests:get-exitcode)
}

:permissions() {
    ls -lR "$1" | sed "s@$1@@" | awk '{print $1, $2, $3, $4}'
}
