#!/bin/bash

if [ "$#" -ne 1 ] || ! [ -e "$1" ]; then
    echo "Script takes one argument, which is a package generator file name"
fi

if [[ $EUID -ne 0 ]]; then
   echo "This script must be run as root"
   exit 1
fi

DIR=$(mktemp -d)
OUTDIR=$(mktemp -d)
ARCH=$(uname -m)

function cleanup {
    rm -rf $DIR $OUTDIR
}
trap cleanup EXIT

echo "Parsing PackageGenerator and downloading files"
pkgenconvert -in $1 -dir $DIR -arch $ARCH || { echo "Build prep failed"; exit 1; }

#currently using alpine docker containers
apk add --no-cache $(cat $DIR/.builddeps.list)  || { echo "Installation of build dependencies failed"; exit 2; }

echo "Done preparing"

#run build in a subshell
echo "Starting build. . . "
(
    cd $DIR
    bash script.sh || { echo "Build failed"; exit 3; }
)
echo "Build complete"

echo "Tarring outputs"
for pkg in `cat $DIR/.pkglist`; do
    tar -cvf $OUTDIR/$pkg.tar.xz -C $DIR/out/$pkg .
done

echo "Uploading outputs"
for pkg in `cat $DIR/.pkglist`; do
    echo $(curl --upload-file $OUTDIR/$pkg.tar.xz https://transfer.sh/$pkg.tar.xz)
done

echo Done
