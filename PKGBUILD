pkgname=gunter
pkgver=1.2e45266
pkgrel=1
pkgdesc="simple configuration system"
url="https://github.com/reconquest/gunter"
arch=('i686' 'x86_64')
license=('GPL')
makedepends=('go')

source=("git://github.com/reconquest/gunter.git")
md5sums=('SKIP')
backup=()

pkgver() {
    cd "${pkgname}"
    echo $(git rev-list --count master).$(git rev-parse --short master)
}

build() {
    cd "$srcdir/$pkgname"

    rm -rf "$srcdir/.go/src"

    mkdir -p "$srcdir/.go/src"

    export GOPATH=$srcdir/.go

    mv "$srcdir/$pkgname" "$srcdir/.go/src/"

    cd "$srcdir/.go/src/gunter/"
    ln -sf "$srcdir/.go/src/gunter/" "$srcdir/$pkgname"

    go get
}

package() {
    mkdir -p "$pkgdir/var/gunter/templates/"
    mkdir -p "$pkgdir/etc/gunter/config"

    mkdir -p "$pkgdir/usr/bin"

    cp "$srcdir/.go/bin/$pkgname" "$pkgdir/usr/bin"
}
