tests:put config <<CONFIG
CONFIG

tests:make-tmp-dir templates/blah.template

tests:ensure :gunter -t templates -d target -c config -l log

tests:assert-test -d target/blah
tests:assert-test ! -d target/blah.template
