#!/bin/bash

cd $(dirname ${0})/..

rm -rf releases
mkdir releases

cd tcpstun

export CGO_ENABLED=0
export NAME="tcpstun"
export VERSION="$(grep 'Version' main.go | grep -Eo '[0-9]\.[0-9]\.[0-9]')"

# Windows
export GOOS=windows
export GOARCH=amd64
go build -ldflags "-s -w" -o $NAME.exe && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME.exe
export GOARCH=386
go build -ldflags "-s -w" -o $NAME.exe && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME.exe

# Linux
export GOOS=linux
export GOARCH=amd64
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME
export GOARCH=386
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME
export GOARCH=arm
export GOARM=7
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME
export GOARCH=arm64
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME

# Mac
export GOOS=darwin
export GOARCH=amd64
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME
export GOARCH=386
go build -ldflags "-s -w" -o $NAME && zip $NAME-$GOOS-$GOARCH-v$VERSION.zip $NAME

mv ${NAME}-*-v$VERSION.zip ../releases
rm -f ${NAME} ${NAME}.exe

