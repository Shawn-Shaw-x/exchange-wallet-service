#!/bin/bash
# in git.bash run

function exit_if() {
    extcode=$1
    msg=$2
    if [ $extcode -ne 0 ]
    then
        if [ "msg$msg" != "msg" ]; then
            echo $msg >&2
        fi
        exit $extcode
    fi
}

# 打印 GOPATH
echo "GOPATH: $GOPATH"

# 检查 protoc-gen-go 是否安装
if ! command -v protoc-gen-go &> /dev/null; then
    echo 'No plugin for golang installed, skip the go installation' >&2
    echo 'try go install google.golang.org/protobuf/cmd/protoc-gen-go@latest' >&2
    exit 1
fi

# 检查 protoc-gen-go-grpc 是否安装
if ! command -v protoc-gen-go-grpc &> /dev/null; then
    echo 'No plugin for golang installed, skip the go installation' >&2
    echo 'try go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest' >&2
    exit 1
fi

# 检查 protoc 是否安装
if ! command -v protoc &> /dev/null; then
    echo "Error: protoc not found. Please install Protocol Buffers compiler." >&2
    exit 1
fi

echo "Compiling go interfaces..."

# 确保 protobuf 目录存在并且包含 .proto 文件
PROTO_DIR="../protobuf"
if [ ! -d "$PROTO_DIR" ]; then
    echo "Error: Directory $PROTO_DIR does not exist." >&2
    exit 1
fi

PROTO_FILES=("$PROTO_DIR"/*.proto)
if [ ! -e "${PROTO_FILES[0]}" ]; then
    echo "Error: No .proto files found in $PROTO_DIR" >&2
    exit 1
fi

# 编译 proto 文件
protoc -I "$PROTO_DIR" \
    --go_out=. \
    --go-grpc_out=require_unimplemented_servers=false:. \
    "$PROTO_DIR"/*.proto

exit_if $? "Failed to compile proto files"
echo "Done"