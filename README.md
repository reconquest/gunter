# Gunter

![gunter](https://cloud.githubusercontent.com/assets/8445924/10263600/b4c470e6-69e3-11e5-9084-930c70a8570f.png)

Gunter is a configuration system which was created with KISS (Keep It
Short and Simple) principle in mind.

Gunter takes a files and directories from the templates directory,
takes a configuration data from the configuration file written in TOML language,
and then create directories with the same names (without `.template` suffix, if
exists), renders `*.template` files via *Go template engine*,
and puts result to destination directory.

Non-template files and directories will by just copied to destination directory.

Of course, **gunter** will save file permissions including file owner uid/gid of
the copied files and directories.

## Installation

There is
[PKGBUILD](https://raw.githubusercontent.com/reconquest/gunter/pkgbuild/PKGBUILD)
file for the *Arch Linux* users.

On other distros **gunter** can be installed via `go get`:

```
go get github.com/reconquest/gunter
```

## Usage

### Options

- `-t <tpl>` - Set source templates directory (default:
    `/var/gunter/templates`).

- `-c <config>` - Set source file with configuration data (default:
    `/etc/gunter/config`).

    The contents of the configuraion file, which must be written in TOML
    language, will be used to render every template file.

- `-d <dir>` - Set destination directory, where rendered template files and
    directories will be saved. (default: `/`)

    After running **gunter**, resulting files and directories will be copied
    to destination directory with the same paths.

- `-r` - "Dry Run" mode. In this mode, **gunter** will create the temporary
    directory, print location, use it as destination directory and will not
    overwrite any system files.

    Very useful for debugging time.

- `-l <log>` - Write overwritten files into specified log. Works in dry-run
   mode too.

- `-p <dir>` - Set directory with plugins (default: `/usr/lib/gunter/plugins/`)


### Templates

All template files should be written in *Go template engine* syntax and have
the `.template` suffix in name.

Read more about syntax:
[http://golang.org/pkg/html/template/](http://golang.org/pkg/html/template/)

If template file uses variables, which does not present in configuration
file, gunter will exit with 1.

#### Tip

Go template engine do not have nice way to suppress new lines before
tags like `{{ end }}`
([github issue](https://github.com/golang/go/issues/9969)).
And template like this:
```
{{ range $item := .Data.Items }}
	{{ $item }}
{{ end }}
```

Will be rendered to:
```
	item1

	item2

	item3
```

And it is not ok, **gunter** can fix this trouble using
`{{ - end }}` tag instead of `{{ end }}`, and template like this:
```
{{ range $item := .Data.Items }}
	{{ $item }}
{{ - end }}
```
*dash symbol prepended before 'end'*

Will be rendered to:
```
	item1
	item2
	item3
```

### Arbitrary example

For example, there are two services `foo` and `bar`, that requires
configuration files, which are located in `/etc/foo/main.conf` and
`/etc/bar.conf`.

So should create template files with filename suffix `.template` in template
directory, for template directory, will be used `/var/templates/`. In this case
it should `/etc/foo/main.conf.template` and `/etc/bar.conf.template`.

Service `foo` should be configured like this:
```
some_persistent_option = 1
host = {{ .Foo.Host }}
port = {{ .Foo.Port }}
another_persistent_option = 1
```

And service `bar` like this:
```
BarUpstream: [ {{ range $ip := .Bar.Addresses }}
        {{ $ip }}
    {{ - end }}
]
```

As configuration file will be used `/etc/superconf`, create it with contents
like as following:
```
# describe foo service
[Foo]
Host = "node0.fo.o"
Port = 80

# and bar service
[Bar]
Addresses = [ "node0.ba.r", "node1.ba.r", "node2.ba.r" ]
```

Run **gunter** "Dry Run" mode passing `-r` flag.
```
gunter -t /var/templates/ -c /etc/superconf -r`
```

After running **gunter** will print directory where all rendered files and
directories are stored:
```
2015/05/22 09:35:30 configuration files are saved into temporary directory
/tmp/gunter281738087/
```

So, structure of `/tmp/gunter281738087/` will be the same as `/var/templates/`
with all file permissions (including file owner uid/gid):

```
/tmp/gunter281738087
└── etc
    ├── bar.conf
    └── foo
        └── main.conf

2 directories, 2 files
```

`/tmp/gunter281738087/etc/foo/main.conf` will have contents:
```
some_persistent_option = 1
host = node0.fo.o
port = 80
another_persistent_option = 1
```

and `/tmp/gunter281738087/etc/bar.conf`:
```
BarUpstream: [
        node0.ba.r
        node1.ba.r
        node2.ba.r
]
```

If result configuration files are ok, **gunter** can be invoked in "normal"
mode and install all configurations to root system directory.
```
gunter -t /var/templates/ -c /etc/superconf
```

After that, all configuration files are directory will be copied to
destination directory and services `foo` and `bar` can start using compiled
configuration data.

### Plugins

Since 2.0 (and Go 1.8) gunter supports plugins, they should be located in
`/usr/lib/gunter/plugins` directory (it can be changed using `-p` flag),
see documentation for Go plugins: https://beta.golang.org/pkg/plugin/)

Plugins should have exported variable `Exports` with type `template.FuncMap`,
that map will be merged against default template functions map and will be
passed to templates compiler and your function will be accessible in templates.

See testcase as example:
https://github.com/reconquest/guntalina/blob/master/tests/testcases/load-plugins.test.sh
