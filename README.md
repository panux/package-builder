# package-builder (WIP)
A tool to create panux packages

[Packages here](https://github.com/panux/packages-main)

## [Docker container](https://hub.docker.com/r/panux/package-builder/)
Example usage:
```
git clone https://github.com/panux/packages-main.git
sudo docker run --rm -v $(realpath packages-main):/build panux/package-builder /build/busybox.pkgen
```
Resulting package will be uploaded to [transfer.sh](https://transfer.sh/) and the link will be printed.
