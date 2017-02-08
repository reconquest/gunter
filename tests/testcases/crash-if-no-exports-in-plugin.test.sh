tests:put plugin.go <<PLUGIN
package main

PLUGIN

tests:make-tmp-dir plugins
tests:ensure go build -buildmode=plugin -o plugins/plugin.so plugin.go

tests:put config <<CONFIG
a = 1
CONFIG

tests:put templates/a.template <<X
stub
X

tests:not tests:ensure :gunter -t templates -d target -c config -l log -p plugins/
tests:assert-stderr "can't lookup Exports variable"
