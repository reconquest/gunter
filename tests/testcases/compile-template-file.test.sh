tests:put config <<CONFIG
str = "string"
int = 1
CONFIG

tests:put templates/a.template <<X
str -> {{ .str }}
int -> {{ .int }}
X

tests:ensure :gunter -t templates -d target -c config -l log

tests:assert-no-diff target/a <<X
str -> string
int -> 1
X
