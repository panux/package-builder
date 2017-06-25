#!/bin/bash

if [ "$#" -ne 1 ] || ! [ -e "$1" ]; then
    echo "Script takes one argument, which is a package generator file name"
fi

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

DIR=$(mktemp -d)

function cleanup {
    rm -rf $DIR
}
trap cleanup EXIT

echo "Parsing PackageGenerator and downloading files"
pkgenconvert -in $1 -dir $DIR

#currently using alpine docker containers
apk add --no-cache $(cat $DIR/.builddeps.list)

echo "Done preparing"

#run build in a subshell
echo "Starting build. . . "
(
    cd $DIR
    bash script.sh
)
echo "Build complete"

echo "Tarring output"
OUT=$(mktemp).tar.xz
tar -cvf $OUT -C $DIR/out .

UPLOAD=$(curl --upload-file $OUT https://transfer.sh/pkg.tar.xz)

echo $UPLOAD
