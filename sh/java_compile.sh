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

if [ ! -f /bin/protoc-gen-grpc-java ]
then
    echo 'No plugin for java installed, skip the java installation' >&2
    echo 'try go get https://repo1.maven.org/maven2/io/grpc/protoc-gen-grpc-java/1.9.1/protoc-gen-grpc-java-1.9.1-linux-x86_64.exe' >&2
else
    echo Compiling java interfaces...
    mkdir -p java
    #export GO_PATH=$GOPATH
    #export GOBIN=$GOPATH/bin
    #export PATH=$PATH:$GOPATH/bin

    protoc -I ./ --java_out=./java --grpc-java_out=./java coincorerpc/*.proto

    exit_if $?
    echo Done
fi
