#!/bin/bash

if [ "$#" -ne 2 ]; then
    echo "Script takes two arguments - package generator file name and output directory"
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
pkgenconvert -in $1 -out $DIR/Makefile || { echo "Build prep failed"; exit 1; }
cat $DIR/Makefile

#run build
echo "Starting build. . . "
make -C $DIR -j 10 all || { echo "Build failed"; exit 3; }
echo "Build complete"

echo "Transferring outputs"
mv $DIR/tars/* $2/

echo Done
