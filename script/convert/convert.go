package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"

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
	debug, err := json.Marshal(pg)
	chk(err)
	log.Println(string(debug))
	chk(pg.InitDir(dir))
}
