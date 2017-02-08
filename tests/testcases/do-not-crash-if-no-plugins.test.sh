tests:put config <<CONFIG
str = "string"
CONFIG

tests:put templates/a.template <<X
str -> {{ .str }}
X

tests:ensure :gunter -t templates -d target -c config -l log

tests:assert-no-diff target/a <<X
str -> string
X
