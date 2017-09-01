pkgenconvert:
	$(MAKE) -C convert

install: pkgenconvert fakename
	$(MAKE) -C convert install
	install -m 0700 build.sh /bin/buildpkg
	rm /bin/uname
	$(MAKE) -C fakename install
	ln -s /usr/bin/uname /bin/uname

fakename:
	git clone https://github.com/panux/fakename.git
