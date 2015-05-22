# Gunter

Gunter is a configuration system which was created with KISS (Keep It
Short and Simple) principle in mind.

Gunter takes a files and directories from the templates directory, takes a
configuration data from the configuration file written in TOML language,
and then create directories with the same names, renders template files via *Go
template engine*, and puts result to destination directory.

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

    All template files should be written in *Go template engine* syntax.
    Read more about syntax:
    [http://golang.org/pkg/html/template/](http://golang.org/pkg/html/template/)

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

### Arbitrary example

For example, there are two services `foo` and `bar`, that requires
configuration files, which are located in `/etc/foo/main.conf` and
`/etc/bar.conf`.

So should create template files with the same paths in template directory, for
template directory, will be used `/var/templates/`.

Service `foo` should be configured like this:
```
some_persistent_option = 1
host = {{ .Foo.Host }}
port = {{ .Foo.Port }}
another_persistent_option = 1
```

And service `bar` like this:
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
SomeUpstream {

        node0.ba.r

        node1.ba.r

        node2.ba.r

}
```

If result configuration files are ok, **gunter** can be invoked in "normal"
mode and install all configurations to root system directory.
```
gunter -t /var/templates/ -c /etc/superconf
```

After that, all configuration files are directory will be copied to
destination directory and services `foo` and `bar` can start using compiled
configuration data.
