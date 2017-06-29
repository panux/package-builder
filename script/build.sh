#!/bin/bash

if [ "$#" -ne 1 ] || ! [ -e "$1" ]; then
    echo "Script takes one argument, which is a package generator file name"
fi

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

DIR=$(mktemp -d)
ARCH=$(uname -m)

function cleanup {
    rm -rf $DIR
}
# trap cleanup EXIT

echo "Parsing PackageGenerator and downloading files"
pkgenconvert -in $1 -out $DIR/Makefile -arch $ARCH || { echo "Build prep failed"; exit 1; }
cat $DIR/Makefile

#run build
echo "Starting build. . . "
make -C $DIR -j 30 all || { echo "Build failed"; exit 3; }
echo "Build complete"

echo "Uploading outputs"
for tar in `ls $DIR/tars`; do
    echo $tar:
    tar -tf $DIR/tars/$tar
    echo $(curl --upload-file $DIR/tars/$tar https://transfer.sh/$tar)
done

echo Done
