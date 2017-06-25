pkgenconvert:
	$(MAKE) -C convert

install: pkgenconvert
	$(MAKE) -C convert install
	install -m 0700 build.sh /bin/buildpkg
