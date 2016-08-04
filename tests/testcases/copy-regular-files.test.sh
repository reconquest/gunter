tests:put config <<CONFIG
str = "string"
int = 1
CONFIG

tests:make-tmp-dir templates/dir
tests:make-tmp-dir -p templates/dir2/subdir

tests:put-string templates/dir/1 'dir/1'
tests:put-string templates/dir2/2 'dir2/2'
tests:put-string templates/dir2/subdir/3 'dir2/subdir/3'

tests:put templates/a.template <<X
str -> {{ .str }}
int -> {{ .int }}
X

tests:ensure :gunter -t templates -d target -c config -l log

tests:assert-no-diff target/a <<X
str -> string
int -> 1
X

tests:assert-no-diff target/dir/1 <<X
dir/1
X

tests:assert-no-diff target/dir2/2 <<X
dir2/2
X

tests:assert-no-diff target/dir2/subdir/3 <<X
dir2/subdir/3
X
