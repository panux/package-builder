package panuxpackager

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
	"gopkg.in/yaml.v2"
)

//RawPackageGenerator represents a PackageGenerator parsed from YAML
type RawPackageGenerator struct {
	Name              string
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
	Names             []string
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
	for i, v := range r.Dependencies {
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
	pg.Names = strings.Split(r.Name, ",")
	for i, v := range pg.Names {
		pg.Names[i] = strings.TrimSpace(v)
	}
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
	for _, v := range pg.Names {
		err = os.Mkdir(filepath.Join(outpath, v), os.ModePerm)
		if err != nil {
			return err
		}
	}
	//Download sources (in parallell)
	cmpl := make(chan error)
	for _, v := range pg.Sources {
		chk := func(err error) {
			if err != nil {
				panic(err)
			}
		}
		getaddr := v
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
			switch getaddr.Scheme {
			case "http":
				panic(errors.New("Insecure HTTP not supported in package files"))
			case "https":
				g, err := http.Get(getaddr.String())
				chk(err)
				defer func() { chk(g.Body.Close()) }()
				faddr := filepath.Join(srcpath, filepath.Base(gpath))
				f, err := os.OpenFile(faddr, os.O_CREATE|os.O_WRONLY, 0644)
				chk(err)
				defer func() { chk(f.Close()) }()
				_, err = io.Copy(f, g.Body)
				chk(err)
			case "git":
				destpath := filepath.Join(srcpath, filepath.Base(getaddr.Path))
				cloneaddr := *getaddr
				cloneaddr.RawQuery = ""
				cmd := exec.Command("git", "clone", cloneaddr.String(), destpath)
				chk(cmd.Run())
				tag := getaddr.Query().Get("tag")
				if tag != "" {
					cmd = exec.Command("git", "-C", destpath, "checkout", tag)
					chk(cmd.Run())
				}
			}
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
	for _, v := range pg.Names {
		//Write package info to file
		pkginfo := struct {
			Name         string
			Version      string
			Dependencies []string
		}{
			Name:         v,
			Version:      pg.Version.String(),
			Dependencies: pg.Dependencies,
		}
		o, err := yaml.Marshal(pkginfo)
		if err != nil {
			return err
		}
		err = ioutil.WriteFile(filepath.Join(outpath, v, ".pkginfo"), o, 0700)
		if err != nil {
			return err
		}
	}
	//Write build-dependencies to file
	bdpath := filepath.Join(path, ".builddeps.list")
	err = ioutil.WriteFile(bdpath, []byte(strings.Join(pg.BuildDependencies, "\n")), 0700)
	if err != nil {
		return err
	}
	//Write package output list to file
	plistpath := filepath.Join(path, ".pkglist")
	err = ioutil.WriteFile(plistpath, []byte(strings.Join(pg.Names, "\n")), 0700)
	if err != nil {
		return err
	}
	return nil
}
