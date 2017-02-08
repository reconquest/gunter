tests:put plugin.go <<PLUGIN
package main

import "text/template"
import "fmt"

var Exports = template.FuncMap{
    "exec_plugin_func": func(str string) string {
        return fmt.Sprintf("Hello from plugin, %s", str)
    },
}
PLUGIN

tests:make-tmp-dir plugins
tests:ensure go build -buildmode=plugin -o plugins/plugin.so plugin.go

tests:put config <<CONFIG
user = "username"
CONFIG

tests:put templates/a.template <<X
user -> {{ .user | exec_plugin_func }}
X

tests:ensure :gunter -t templates -d target -c config -l log -p plugins/

tests:assert-no-diff target/a <<X
user -> Hello from plugin, username
X
