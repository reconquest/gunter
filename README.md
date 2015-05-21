# Gunter

Gunter it is configuration system which created with KISS principle. (Keep It
Short and Simple)

Gunter takes a files and directories from the templates directory, takes a
configuration data from the configuration file written with the TOML language,
creating directories with the same names, compiling template files via *Go
template engine*, and puts that all to destination directory.

Of course, **gunter** saves file permissions including file owner uid/gid of
copied files and directories.

## Installation

There is
[PKGBUILD](https://raw.githubusercontent.com/reconquest/gunter/pkgbuild/PKGBUILD)
file for the *Arch Linux* users.

On other distros **gunter** should be installed via `go get`:

```
go get github.com/reconquest/gunter
```

## Usage

### Options

- `-t <tpl>` - Set specified templates directory (default:
    `/etc/gunter/templates`).

    All template files should has valid *Go template engine* syntax.
    Read more about syntax:
    [http://golang.org/pkg/html/template/](http://golang.org/pkg/html/template/)

- `-c <config>` - Set specified configuration file (default:
    `/etc/gunter/config`).

    The contents of the configuraion file, which must be written with TOML
    language, will be passed to every template file.

- `-d <dir>` - Use specified directory path as destination. (default: `/`)

    After running **gunter**, resulting files and directories will be copied
    to destination directory with the same paths.

- `-r` - "Dry Run" mode. In this mode, **gunter** creates the temporary
    directory, print location, and use it as destination directory. Very useful
    for debugging time.

### Arbitary example

For example, needs to configure services `foo` and `bar` which configuration
files are located in `/etc/foo/main.conf` and `/etc/bar.conf`.

So should create template files with the same paths in template directory, for
template directory, will be used `/etc/templates/`.

Create template `/etc/templates/etc/foo/main.conf` with some contents
like as following:
```
some_persistent_option = 1
host = {{ .Foo.Host }}
port = {{ .Foo.Port }}
another_persistent_option = 1
```

And `/etc/bar.conf`:
```
SomeUpstream {
    {{ range $ip := .Bar.Addresses }}
        {{ $ip }}
    {{ end }}
}
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

For debugging run **gunter** with all that options in "Dry Run" mode:
```
gunter -t /etc/templates/ -c /etc/superconf -r`
```

After running will be printed message:
```
2015/05/22 09:35:30 configuration files are saved into temporary directory
/tmp/gunter281738087/
```

So, structure of `/tmp/gunter281738087/` will be same as `/etc/templates/` with
all file permissions (including file owner uid/gid):

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
SomeUpstream {

        node0.ba.r

        node1.ba.r

        node2.ba.r

}
```

Everything is okay, now **gunter** may run in "normal" mode and install all
configs to root system directory.
```
gunter -t /etc/templates/ -c /etc/superconf
```

After this step, all configuration files are directory will be copied to
destination directory.
