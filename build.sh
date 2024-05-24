#!/bin/sh
cd -P -- "$(dirname -- "$0")"

VERSION=$(git tag -l --points-at HEAD)
if [ -z "$VERSION" ]
then
  VERSION="$(git rev-parse --short HEAD)-$(date -u '+%Y.%m.%d-%H:%M:%S')"
else
  VERSION="v${VERSION}"
fi

TARGET_OS=${2%-*}
TARGET_ARCH=${2#*-}

TARGET_EXT=""
if [ "$TARGET_OS" == "windows" ]; then TARGET_EXT=".exe"; fi

case "$1" in
  release)
    echo "---> Build Lit3D-KUKA-C3-Gate for ${TARGET_OS}-${TARGET_ARCH} to build/lit3d-kuka-c3-gate${TARGET_EXT}"
    eval 'GOOS=${TARGET_OS} GOARCH=${TARGET_ARCH} go build -mod=vendor -ldflags "-w -s -X main.version=${VERSION}" -o ./build/lit3d-kuka-c3-gate${TARGET_EXT} ./cmd/kuka-c3-gate'
    cp ./bots.json ./build/bots.json
    ;;
  run)
    echo "---> Running Lit3D-KUKA-C3-Gate"
    eval 'go run -mod=vendor -tags=dev -ldflags "-X main.version=${VERSION}" ./cmd/kuka-c3-gate -v -ui'
    ;;
  *)
    echo "Incorrect build target name" >&2
    exit 1
    ;;
esac
