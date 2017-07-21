# package-builder (WIP) [![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/panux/panux-dev)[![Docker Pulls](https://img.shields.io/docker/pulls/panux/package-builder.svg)](https://hub.docker.com/r/panux/package-builder/)[![GoDoc](https://godoc.org/github.com/panux/package-builder?status.svg)](https://godoc.org/github.com/panux/package-builder)
A tool to create panux packages

[Packages here](https://github.com/panux/packages-main)

## [Docker container](https://hub.docker.com/r/panux/package-builder/)
Example usage:
```
git clone https://github.com/panux/packages-main.git
mkdir out
sudo docker run --rm -v $(realpath packages-main):/build -v $(realpath out):/out panux/package-builder /build/busybox.pkgen /out
```
Resulting package will be moved to out

In addition, this repository exposes a Go package which allows scripting of package file parsing/preprocessing. The package allows .pkgen files to be parsed from a byte array or file, preprocessed, and then be converted into a Makefile written to an io.Writer. The contents of the package file can also be accessed through the intermediary structs.
