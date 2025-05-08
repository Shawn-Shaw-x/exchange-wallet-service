#!/bin/bash

mkdir -p python/savourrpc
touch python/savourrpc/__init__.py

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

protofiles=$(find . -name '*.proto')

echo Compiling python interfaces...

python3 -m grpc_tools.protoc -I ./ \
       --python_out=python/ \
       --grpc_python_out=python/ \
       $protofiles
exit_if $?

if [ yes`which protoc-gen-grpclib_python` != yes ]; then
    python3 -m grpc_tools.protoc -I ./ \
           --grpclib_python_out=python/ \
           $protofiles

    exit_if $?
else
    echo 'No plugin for grpclib installed, skip the go installation' >&2
fi
echo Done

if [ ! -f $GOPATH/bin/protoc-gen-go ]
then
    echo 'No plugin for golang installed, skip the go installation' >&2
    echo 'try go get github.com/golang/protobuf/protoc-gen-go' >&2
else
    echo Compiling go interfaces...
    mkdir -p go
    export GO_PATH=$GOPATH
    export GOBIN=$GOPATH/bin
    export PATH=$PATH:$GOPATH/bin
    protoc -I ./ \
           --go_out=plugins=grpc:go \
           $protofiles
    exit_if $?
    echo Done
fi
