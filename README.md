# package-builder (WIP) [![Gitter chat](https://badges.gitter.im/gitterHQ/gitter.png)](https://gitter.im/panux/panux-dev)
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
