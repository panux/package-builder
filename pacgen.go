package panuxpackager

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hashicorp/go-version"
	"github.com/panux/encoding-sh"
	"gopkg.in/yaml.v2"
)

//RawPackageGenerator represents a PackageGenerator parsed from YAML
type RawPackageGenerator struct {
	SrcPath           string
	Packages          pkgmap
	OneShell          bool
	Tools             []string
	Version           string
	Sources           []string
	Script            []string
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

//ParseFile parses a RawPackageGenerator from data in a file
func ParseFile(file string) (pg RawPackageGenerator, err error) {
	dat, err := ioutil.ReadFile(file)
	if err != nil {
		return RawPackageGenerator{}, err
	}
	pg, err = ParseRaw(dat)
	if err != nil {
		return RawPackageGenerator{}, err
	}
	pg.SrcPath = file
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
	SrcPath           string
	DestPath          string
	Pkgs              pkgmap
	OneShell          bool
	Tools             []*Tool
	Version           *version.Version
	Sources           []*url.URL
	Script            string
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

type pkgmap map[string]*pkg
type pkg struct {
	Dependencies []string
}

//Preprocess runs preprocessing steps on the RawPackageGenerator in order to convert it to a PackageGenerator
func (r RawPackageGenerator) Preprocess() (pg PackageGenerator, err error) {
	pg.SrcPath = r.SrcPath
	pg.OneShell = r.OneShell
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
	nval := []string{}
	pg.Pkgs = make(pkgmap)
	for x, y := range r.Packages {
		pg.Pkgs[x] = new(pkg)
		if y != nil && y.Dependencies != nil {
			pg.Pkgs[x].Dependencies = make([]string, len(y.Dependencies))
			for i, v := range y.Dependencies {
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
				pg.Pkgs[x].Dependencies[i] = buf.String()
			}
		} else {
			pg.Pkgs[x].Dependencies = nval
		}
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

func dirGen(path string, w io.Writer) error {
	dir := filepath.Dir(path)
	if dir == "." {
		dir = ""
	}
	_, err := fmt.Fprintf(w, "%s: %s\n\tmkdir %s\n\n", path, dir, path)
	return err
}

//GenSetupMake generates the Makefile to create the source package
func (pg PackageGenerator) GenSetupMake(w io.Writer) error {
	_, err := fmt.Fprintln(w, "all: src.tar.gz\n\nsrc.tar.gz: sources destinations pkginfo tars\n\ttar -cvf src.tar.gz -C . . --exclude makefile")
	if err != nil {
		return err
	}
	//Write package info strings
	version := pg.Version.String()
	for n, v := range pg.Pkgs {
		pkginfo := struct {
			Name         string
			Version      string
			Dependencies []string
		}{
			Name:         n,
			Version:      version,
			Dependencies: v.Dependencies,
		}
		dat, err := sh.Encode(pkginfo)
		if err != nil {
			return err
		}
		ystr := string(dat)
		_, err = fmt.Fprintf(w, "define %s_pkginfo = \n%s\nendef\n", strings.Replace(n, "-", "_", -1), ystr)
		if err != nil {
			return err
		}
	}
	//Write directory structure generation
	err = dirGen("src", w)
	if err != nil {
		return err
	}
	err = dirGen("out", w)
	if err != nil {
		return err
	}
	err = dirGen("tars", w)
	if err != nil {
		return err
	}
	//Sources
	srcs := make([]string, len(pg.Sources))
	for i, v := range pg.Sources {
		fname := filepath.Base(v.Path)
		switch v.Scheme {
		case "http":
			return errors.New("Insecure HTTP not supported for package sources")
		case "https":
			_, err = fmt.Fprintf(w, "\nsrc/%s: src\n\tcurl %s > src/%s\n\n", fname, v.String(), fname)
			if err != nil {
				return err
			}
		case "git":
			if filepath.Ext(fname) == ".git" {
				fname = fname[:len(fname)-5]
			}
			u := *v
			u.RawQuery = ""
			_, err = fmt.Fprintf(w, "\nsrc/%s: src\n\tgit clone %s src/%s\n\tgit -C src/%s checkout %s\n\n", fname, u.String(), fname, fname, v.Query().Get("checkout"))
			if err != nil {
				return err
			}
		case "file":
			dir, _ := filepath.Split(pg.SrcPath)
			srcpath := filepath.Join(dir, v.Path)
			destpath := fmt.Sprintf("%s/src/%s", pg.DestPath, fname)
			err = func() error {
				sf, err := os.Open(srcpath)
				if err != nil {
					return err
				}
				defer sf.Close()
				df, err := os.OpenFile(destpath, os.O_WRONLY|os.O_CREATE, 0600)
				if err != nil {
					return err
				}
				defer df.Close()
				_, err = io.Copy(df, sf)
				if err != nil {
					return err
				}
				return nil
			}()
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unsupported source scheme %s://", v.Scheme)
		}
		srcs[i] = "src/" + fname
	}
	_, err = fmt.Fprintf(w, "sources: %s\n\n", strings.Join(srcs, " "))
	if err != nil {
		return err
	}
	//Destination directories
	dests := make([]string, len(pg.Pkgs))
	i := 0
	for n := range pg.Pkgs {
		dests[i] = "out/" + n
		i++
		err = dirGen("out/"+n, w)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprintf(w, "destinations: %s\n\n", strings.Join(dests, " "))
	if err != nil {
		return err
	}
	infos := make([]string, len(pg.Pkgs))
	i = 0
	//Write out package info files
	for n := range pg.Pkgs {
		_, err = fmt.Fprintf(w, "export %s_pkginfo\nout/%s/.pkginfo: out/%s\n\techo \"$$%s_pkginfo\" > out/%s/.pkginfo\n\n", strings.Replace(n, "-", "_", -1), n, n, strings.Replace(n, "-", "_", -1), n)
		if err != nil {
			return err
		}
		infos[i] = fmt.Sprintf("out/%s/.pkginfo", n)
		i++
	}
	_, err = fmt.Fprintf(w, "pkginfo: %s\n", strings.Join(infos, " "))
	if err != nil {
		return err
	}
	return nil
}

//GenMake generates the build Makefile
func (pg PackageGenerator) GenMake(w io.Writer) error {
	var err error
	//BuildDependencies
	if len(pg.BuildDependencies) > 0 {
		_, err = fmt.Fprintf(w, "builddeps: \n\tapk add --no-cache %s\n\ttouch builddeps\n\n", strings.Join(pg.BuildDependencies, " "))
		if err != nil {
			return err
		}
	} else {
		_, err = fmt.Fprintln(w, "builddeps: ")
		if err != nil {
			return err
		}
	}
	//Tar packages
	for n := range pg.Pkgs {
		_, err = fmt.Fprintf(w, "tars/%s.tar.xz: out/%s/.pkginfo build\n\ttar -cf tars/%s.tar.xz -C out/%s .\n\n", n, n, n, n)
		if err != nil {
			return err
		}
	}
	//Generate main target
	pkgtargs := make([]string, len(pg.Pkgs))
	i := 0
	for n := range pg.Pkgs {
		pkgtargs[i] = fmt.Sprintf("tars/%s.tar.xz", n)
		i++
	}
	_, err = fmt.Fprintf(w, "all: %s\n\n", strings.Join(pkgtargs, " "))
	if err != nil {
		return err
	}
	//Build script
	pr := ""
	if pg.OneShell {
		pr = ".ONESHELL:\n"
	}
	_, err = fmt.Fprintf(w, "%sbuild: builddeps\n\t%s\n\ttouch build\n\n", pr, strings.Join(strings.Split(pg.Script, "\n"), "\n\t"))
	if err != nil {
		return err
	}
	return nil
}

//SetupDir downloads sources into a directory and loads in an appropriate Makefile
func (pg PackageGenerator) SetupDir(dir string) error {
	pg.DestPath = dir
	err := func() error {
		mf, err := os.OpenFile(dir+"/makefile", os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer mf.Close()
		return pg.GenSetupMake(mf)
	}()
	if err != nil {
		return err
	}
	err = func() error {
		mf, err := os.OpenFile(dir+"/Makefile", os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		defer mf.Close()
		return pg.GenMake(mf)
	}()
	if err != nil {
		return err
	}
	cmd := exec.Command("make", "-C", dir, "-j10", "-f", "makefile", "all")
	d, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}
	if !cmd.ProcessState.Success() {
		return errors.New("Make failed")
	}
	fmt.Println(string(d))
	err = os.Remove(dir + "/makefile")
	if err != nil {
		return err
	}
	return nil
}

//GenPkgSrc generates a tar file with package source
func (pg PackageGenerator) GenPkgSrc(w io.Writer) (err error) {
	dd, err := exec.Command("mktemp", "-d").Output()
	if err != nil {
		return err
	}
	dir := strings.TrimSpace(string(dd))
	defer os.RemoveAll(dir)
	err = pg.SetupDir(dir)
	if err != nil {
		return err
	}
	f, err := os.Open(dir + "/src.tar.gz")
	if err != nil {
		return err
	}
	defer func() {
		e := f.Close()
		if err == nil {
			err = e
		}
	}()
	n, err := io.Copy(w, f)
	if err != nil {
		return err
	}
	log.Printf("Copied %d bytes\n", n)
	return nil
}
