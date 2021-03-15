#!/bin/sh
cd `dirname "$0"`
n="influsender"
alist="386 amd64 arm arm64 mips mips64"
rm "$n"_*
for ARCH in $alist; do
    echo "Building binary for $ARCH"
    env GOOS=$OS GOARCH=$ARCH go build -o "$n"_"$ARCH" $n.go
    cp "$n"_"$ARCH" "$n"_"$ARCH"_compressed
    [ -n "$(which upx)" ] && echo "Compressing binary ""$n"_"$ARCH" && upx "$n"_"$ARCH"_compressed
done
