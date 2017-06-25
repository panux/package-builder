package main

import (
	"flag"
	"io/ioutil"

	"../.."
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var dir string
	var infile string
	var arch string
	flag.StringVar(&infile, "in", "", "input file to generate from")
	flag.StringVar(&dir, "dir", "", "directory to generate into")
	flag.StringVar(&arch, "arch", "x86_64", "cpu architecture")
	flag.Parse()
	dat, err := ioutil.ReadFile(infile)
	chk(err)
	r, err := panuxpackager.ParseRaw(dat)
	chk(err)
	r.Arch = arch
	pg, err := r.Preprocess()
	chk(err)
	chk(pg.InitDir(dir))
}
