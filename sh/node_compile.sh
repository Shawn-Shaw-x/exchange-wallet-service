#!/bin/bash

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

echo Compiling node interfaces...
mkdir -p node

#protoc -I ./ --java_out=./java --grpc-java_out=./java protobuf/*.proto
protoc --js_out=import_style=commonjs,binary:./node savourrpc/*.proto

exit_if $?
echo Done
