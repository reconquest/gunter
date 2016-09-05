tests:put config <<CONFIG
CONFIG

tests:put templates/a.template <<X
str -> {{ .str }}
int -> {{ .int }}

items:
{{ range .list }}
- {{.}}
{{ end}}
X

tests:ensure :gunter -t templates -d target -c config -l log -n

tests:assert-no-diff target/a <<X
str -> <no value>
int -> <no value>

items:
X
