pkgenconvert:
	$(MAKE) -C convert

install: pkgenconvert uname
	$(MAKE) -C convert install
	install -m 0700 build.sh /bin/buildpkg
	rm /bin/uname
	install -m 0700 uname /bin/uname

gouname:
	git clone https://github.com/jadr2ddude/gouname.git

uname: gouname
	$(MAKE) -C gouname
	mv gouname/uname uname
