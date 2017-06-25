package main

import (
	"flag"
	"fmt"
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
	flag.StringVar(&infile, "in", "", "input file to generate from")
	flag.StringVar(&dir, "dir", "", "directory to generate into")
	flag.Parse()
	dat, err := ioutil.ReadFile(infile)
	chk(err)
	fmt.Println(string(dat))
	r, err := panuxpackager.ParseRaw(dat)
	chk(err)
	fmt.Println(r)
	pg, err := r.Preprocess()
	chk(err)
	chk(pg.InitDir(dir))
}
