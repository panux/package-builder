package panuxpackager

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
	"gopkg.in/yaml.v2"
)

//RawPackageGenerator represents a PackageGenerator parsed from YAML
type RawPackageGenerator struct {
	Tools             []string
	Version           string
	Sources           []string
	Script            []string
	Dependencies      []string
	BuildDependencies []string
	Arch              string
	Data              map[string]interface{}
}

//ParseRaw parses a RawPackageGenerator from data in a []byte
func ParseRaw(in []byte) (pg RawPackageGenerator, err error) {
	err = yaml.Unmarshal(in, &pg)
	if err != nil {
		pg = RawPackageGenerator{}
	}
	return
}

//Tool is a tool for a PackageGenerator
type Tool struct {
	Name         string
	Version      *version.Version
	ToolFuncs    map[string]func() (interface{}, error)
	Dependencies []string
}

var tools map[string]*Tool

//PackageGenerator is a preprocessed package generator
type PackageGenerator struct {
	Tools             []*Tool
	Version           *version.Version
	Sources           []*url.URL
	Script            string
	Dependencies      []string
	BuildDependencies []string
}

func (pg PackageGenerator) toolfuncs() (m map[string]interface{}) {
	m = make(map[string]interface{})
	for _, v := range pg.Tools {
		for n, t := range v.ToolFuncs {
			m[n] = t
		}
	}
	return
}

//Preprocess runs preprocessing steps on the RawPackageGenerator in order to convert it to a PackageGenerator
func (r RawPackageGenerator) Preprocess() (pg PackageGenerator, err error) {
	npg := PackageGenerator{}
	for _, name := range r.Tools {
		tool := tools[name]
		if tool == nil {
			return npg, fmt.Errorf("Tool %q not found", name)
		}
	}
	tf := pg.toolfuncs()
	pg.Version, err = version.NewVersion(r.Version)
	if err != nil {
		return npg, err
	}
	pg.Sources = make([]*url.URL, len(r.Sources))
	for i, v := range r.Sources {
		tmpl, err := template.New("sources").Parse(v)
		if err != nil {
			return npg, err
		}
		tmpl.Funcs(tf)
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, r)
		if err != nil {
			return npg, err
		}
		sstr := buf.String()
		src, err := url.Parse(sstr)
		if err != nil {
			return npg, err
		}
		pg.Sources[i] = src
	}
	pg.BuildDependencies = make([]string, len(r.BuildDependencies))
	for i, v := range r.BuildDependencies {
		tmpl, err := template.New("build_dependencies").Parse(v)
		if err != nil {
			return npg, err
		}
		tmpl.Funcs(tf)
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, r)
		if err != nil {
			return npg, err
		}
		pg.BuildDependencies[i] = buf.String()
	}
	for _, v := range pg.Tools {
		if v.Dependencies != nil {
			pg.BuildDependencies = append(pg.BuildDependencies, v.Dependencies...)
		}
	}
	pg.Dependencies = make([]string, len(r.Dependencies))
	for i, v := range r.BuildDependencies {
		tmpl, err := template.New("dependencies").Parse(v)
		if err != nil {
			return npg, err
		}
		tmpl.Funcs(tf)
		buf := bytes.NewBuffer(nil)
		err = tmpl.Execute(buf, r)
		if err != nil {
			return npg, err
		}
		pg.Dependencies[i] = buf.String()
	}
	stmpl, err := template.New("script").Parse(strings.Join(r.Script, "\n"))
	if err != nil {
		return npg, err
	}
	stmpl.Funcs(tf)
	buf := bytes.NewBuffer(nil)
	err = stmpl.Execute(buf, r)
	if err != nil {
		return npg, err
	}
	pg.Script = buf.String()
	return
}

//InitDir initializes a directory for generating the package
func (pg PackageGenerator) InitDir(path string) error {
	//Make workdir structure
	srcpath := filepath.Join(path, "src")
	outpath := filepath.Join(path, "out")
	err := os.Mkdir(srcpath, os.ModePerm)
	if err != nil {
		return err
	}
	err = os.Mkdir(outpath, os.ModePerm)
	if err != nil {
		return err
	}
	//Download sources (in parallell)
	cmpl := make(chan error)
	for _, v := range pg.Sources {
		chk := func(err error) {
			if err != nil {
				panic(err)
			}
		}
		getaddr := v.String()
		gpath := v.Path
		go func() {
			defer func() { recover() }() //recover any error from termination
			defer func() {
				err := recover()
				if err != nil {
					cmpl <- err.(error)
				} else {
					cmpl <- nil
				}
			}()
			log.Printf("Downloading %s\n", getaddr)
			g, err := http.Get(getaddr)
			chk(err)
			defer func() { chk(g.Body.Close()) }()
			faddr := filepath.Join(srcpath, ""+filepath.Base(gpath))
			f, err := os.OpenFile(faddr, os.O_CREATE|os.O_WRONLY, 0644)
			chk(err)
			defer func() { chk(f.Close()) }()
			_, err = io.Copy(f, g.Body)
			chk(err)
		}()
	}
	n := len(pg.Sources)
	for n > 0 {
		err := <-cmpl
		if err != nil {
			close(cmpl)
			return err
		}
		n--
	}
	//Write script to file
	spath := filepath.Join(path, "script.sh")
	err = ioutil.WriteFile(spath, []byte(pg.Script), 0700)
	if err != nil {
		return err
	}
	//Write dependencies to file
	dpath := filepath.Join(outpath, ".deps.list")
	err = ioutil.WriteFile(dpath, []byte(strings.Join(pg.Dependencies, "\n")), 0700)
	if err != nil {
		return err
	}
	//Write build-dependencies to file
	bdpath := filepath.Join(path, ".builddeps.list")
	err = ioutil.WriteFile(bdpath, []byte(strings.Join(pg.BuildDependencies, "\n")), 0700)
	if err != nil {
		return err
	}
	//Write version to file
	vpath := filepath.Join(outpath, ".version")
	err = ioutil.WriteFile(vpath, []byte(pg.Version.String()), 0700)
	if err != nil {
		return err
	}
	return nil
}
