#!/bin/bash

set -e

function exit_if() {
    extcode=$1
    msg=$2
    if [ $extcode -ne 0 ]; then
        if [ "msg$msg" != "msg" ]; then
            echo "$msg" >&2
        fi
        exit $extcode
    fi
}

echo "🔧 Checking for required protoc plugins..."

# 检查插件是否在 PATH 中
if ! command -v protoc-gen-go >/dev/null 2>&1 || ! command -v protoc-gen-go-grpc >/dev/null 2>&1; then
    echo '❌ Missing protoc plugins for Go:' >&2
    echo '  - protoc-gen-go' >&2
    echo '  - protoc-gen-go-grpc' >&2
    echo '' >&2
    echo '👉 Install them with:' >&2
    echo '  go install google.golang.org/protobuf/cmd/protoc-gen-go@latest' >&2
    echo '  go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest' >&2
    echo '' >&2
    echo '🔁 And make sure $GOBIN or $(go env GOBIN) is in your PATH' >&2
    exit 1
fi

echo "✅ Plugins found, starting compilation..."

# 编译 proto 文件
protoc \
    -I ./ \
    --go_out=./ \
    --go-grpc_out=require_unimplemented_servers=false:./ \
    ./protobuf/*.proto

exit_if $? "❌ protoc compilation failed"

echo "✅ Done generating Go code from proto files"