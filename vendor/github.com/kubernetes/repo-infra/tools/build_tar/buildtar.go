/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// fast tar builder for bazel
package main

import (
	"archive/tar"
	"bufio"
	"compress/bzip2"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/golang/glog"

	"golang.org/x/build/pargzip"
)

func main() {
	var (
		flagfile string

		output      string
		directory   string
		compression string

		files multiString
		tars  multiString
		debs  multiString
		links multiString

		mode  string
		modes multiString

		owner      string
		owners     multiString
		ownerName  string
		ownerNames multiString
	)

	flag.StringVar(&flagfile, "flagfile", "", "Path to flagfile")

	flag.StringVar(&output, "output", "", "The output file, mandatory")
	flag.StringVar(&directory, "directory", "", "Directory in which to store the file inside the layer")
	flag.StringVar(&compression, "compression", "", "Compression (`gz` or `bz2`), default is none.")

	flag.Var(&files, "file", "A file to add to the layer")
	flag.Var(&tars, "tar", "A tar file to add to the layer")
	flag.Var(&debs, "deb", "A debian package to add to the layer")
	flag.Var(&links, "link", "Add a symlink a inside the layer ponting to b if a:b is specified")

	flag.StringVar(&mode, "mode", "", "Force the mode on the added files (in octal).")
	flag.Var(&modes, "modes", "Specific mode to apply to specific file (from the file argument), e.g., path/to/file=0455.")

	flag.StringVar(&owner, "owner", "0.0", "Specify the numeric default owner of all files, e.g., 0.0")
	flag.Var(&owners, "owners", "Specify the numeric owners of individual files, e.g. path/to/file=0.0.")
	flag.StringVar(&ownerName, "owner_name", "", "Specify the owner name of all files, e.g. root.root.")
	flag.Var(&ownerNames, "owner_names", "Specify the owner names of individual files, e.g. path/to/file=root.root.")

	flag.Set("alsologtostderr", "true")

	flag.Parse()

	if flagfile != "" {
		b, err := ioutil.ReadFile(flagfile)
		if err != nil {
			glog.Fatalf("couldn't read flagfile: %v", err)
		}
		cmdline := strings.Split(string(b), "\n")
		flag.CommandLine.Parse(cmdline)
	}

	if output == "" {
		glog.Fatalf("--output flag is required")
	}

	meta := newFileMeta(mode, modes, owner, owners, ownerName, ownerNames)

	tf, err := newTarFile(output, directory, compression, meta)
	if err != nil {
		glog.Fatalf("couldn't build tar: %v", err)
	}
	defer tf.Close()

	for _, file := range files {
		parts := strings.SplitN(file, "=", 2)
		if len(parts) != 2 {
			glog.Fatalf("bad parts length for file %q", file)
		}
		if err := tf.addFile(parts[0], parts[1]); err != nil {
			glog.Fatalf("couldn't add file: %v", err)
		}
	}

	for _, tar := range tars {
		if err := tf.addTar(tar); err != nil {
			glog.Fatalf("couldn't add tar: %v", err)
		}
	}

	for _, deb := range debs {
		if err := tf.addDeb(deb); err != nil {
			glog.Fatalf("couldn't add deb: %v", err)
		}
	}

	for _, link := range links {
		parts := strings.SplitN(link, ":", 2)
		if len(parts) != 2 {
			glog.Fatalf("bad parts length for link %q", link)
		}
		if err := tf.addLink(parts[0], parts[1]); err != nil {
			glog.Fatalf("couldn't add link: %v", err)
		}
	}
}

type tarFile struct {
	directory string

	tw *tar.Writer

	meta     fileMeta
	dirsMade map[string]struct{}

	closers []func()
}

func newTarFile(output, directory, compression string, meta fileMeta) (*tarFile, error) {
	var (
		w       io.Writer
		closers []func()
	)
	f, err := os.Create(output)
	if err != nil {
		return nil, err
	}
	closers = append(closers, func() {
		f.Close()
	})
	w = f

	buf := bufio.NewWriter(w)
	closers = append(closers, func() { buf.Flush() })
	w = buf

	switch compression {
	case "":
	case "gz":
		gzw := pargzip.NewWriter(w)
		closers = append(closers, func() { gzw.Close() })
		w = gzw
	case "bz2", "xz":
		return nil, fmt.Errorf("%q compression is not supported yet", compression)
	default:
		return nil, fmt.Errorf("unknown compression %q", compression)
	}

	tw := tar.NewWriter(w)
	closers = append(closers, func() { tw.Close() })

	return &tarFile{
		directory: directory,
		tw:        tw,
		closers:   closers,
		meta:      meta,
		dirsMade:  map[string]struct{}{},
	}, nil
}

func (f *tarFile) addFile(file, dest string) error {
	dest = strings.TrimLeft(dest, "/")
	dest = filepath.Clean(dest)

	uid := f.meta.getUID(dest)
	gid := f.meta.getGID(dest)
	uname := f.meta.getUname(dest)
	gname := f.meta.getGname(dest)

	dest = filepath.Join(strings.TrimLeft(f.directory, "/"), dest)
	dest = filepath.Clean(dest)

	info, err := os.Stat(file)
	if err != nil {
		return err
	}

	mode := f.meta.getMode(dest)
	// If mode is unspecified, derive the mode from the file's mode.
	if mode == 0 {
		mode = os.FileMode(0644)
		if info.Mode().Perm()&os.FileMode(0111) != 0 {
			mode = os.FileMode(0755)
		}
	}

	header := tar.Header{
		Name:  dest,
		Mode:  int64(mode),
		Uid:   uid,
		Gid:   gid,
		Size:  0,
		Uname: uname,
		Gname: gname,
	}

	if err := f.makeDirs(header); err != nil {
		return err
	}

	switch {
	case info.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("addFile: didn't expect symlink: %s", file)
	case info.Mode()&os.ModeNamedPipe != 0:
		return fmt.Errorf("addFile: didn't expect named pipe: %s", file)
	case info.Mode()&os.ModeSocket != 0:
		return fmt.Errorf("addFile: didn't expect socket: %s", file)
	case info.Mode()&os.ModeDevice != 0:
		return fmt.Errorf("addFile: didn't expect device: %s", file)
	case info.Mode()&os.ModeDir != 0:
		header.Typeflag = tar.TypeDir
		if err := f.tw.WriteHeader(&header); err != nil {
			return err
		}
	default:
		//regular file
		header.Typeflag = tar.TypeReg
		b, err := ioutil.ReadFile(file)
		if err != nil {
			return err
		}
		header.Size = int64(len(b))
		if err := f.tw.WriteHeader(&header); err != nil {
			return err
		}
		if _, err := f.tw.Write(b); err != nil {
			return err
		}
	}
	return nil
}

func (f *tarFile) addLink(symlink, target string) error {
	header := tar.Header{
		Name:     symlink,
		Typeflag: tar.TypeSymlink,
		Linkname: target,
	}
	if err := f.makeDirs(header); err != nil {
		return err
	}
	return f.tw.WriteHeader(&header)
}

func (f *tarFile) addTar(toAdd string) error {
	root := ""
	if f.directory != "/" {
		root = f.directory
	}

	var r io.Reader

	file, err := os.Open(toAdd)
	if err != nil {
		return err
	}
	defer file.Close()
	r = file

	r = bufio.NewReader(r)

	switch {
	case strings.HasSuffix(toAdd, "gz"):
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return err
		}
		r = gzr
	case strings.HasSuffix(toAdd, "bz2"):
		bz2r := bzip2.NewReader(r)
		r = bz2r
	case strings.HasSuffix(toAdd, "xz"):
		return fmt.Errorf("%q decompression is not supported yet", toAdd)
	default:
	}

	tr := tar.NewReader(r)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		header.Name = filepath.Join(root, header.Name)
		if header.Typeflag == tar.TypeDir && !strings.HasSuffix(header.Name, "/") {
			header.Name = header.Name + "/"
		}
		err = f.tw.WriteHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(f.tw, tr); err != nil {
			return err
		}
	}
	return nil
}

func (f *tarFile) addDeb(toAdd string) error {
	return fmt.Errorf("addDeb unimplemented")
}

func (f *tarFile) makeDirs(header tar.Header) error {
	dirToMake := []string{}
	dir := header.Name
	for {
		dir = filepath.Dir(dir)
		if dir == "." || dir == "/" {
			break
		}
		dirToMake = append(dirToMake, dir)
	}
	for i := len(dirToMake) - 1; i >= 0; i-- {
		dir := dirToMake[i]
		if _, ok := f.dirsMade[dir]; ok {
			continue
		}
		dh := header
		dh.Typeflag = tar.TypeDir
		dh.Name = dir + "/"
		if err := f.tw.WriteHeader(&dh); err != nil {
			return err
		}

		f.dirsMade[dir] = struct{}{}
	}
	return nil
}

func (f *tarFile) Close() {
	for i := len(f.closers) - 1; i >= 0; i-- {
		f.closers[i]()
	}
}

func newFileMeta(
	mode string,
	modes multiString,
	owner string,
	owners multiString,
	ownerName string,
	ownerNames multiString,
) fileMeta {
	var meta fileMeta

	if mode != "" {
		i, err := strconv.ParseUint(mode, 8, 32)
		if err != nil {
			glog.Fatalf("couldn't parse mode: %v", mode)
		}
		meta.defaultMode = os.FileMode(i)
	}

	meta.modeMap = map[string]os.FileMode{}
	for _, filemode := range modes {
		parts := strings.SplitN(filemode, "=", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", filemode)
		}
		if parts[0][0] == '/' {
			parts[0] = parts[0][1:]
		}
		i, err := strconv.ParseUint(parts[1], 8, 32)
		if err != nil {
			glog.Fatalf("couldn't parse mode: %v", filemode)
		}
		meta.modeMap[parts[0]] = os.FileMode(i)
	}

	if ownerName != "" {
		parts := strings.SplitN(ownerName, ".", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", ownerName)
		}
		meta.defaultUname = parts[0]
		meta.defaultGname = parts[1]
	}

	meta.unameMap = map[string]string{}
	meta.gnameMap = map[string]string{}
	for _, name := range ownerNames {
		parts := strings.SplitN(name, "=", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q %v", name, parts)
		}
		filename, ownername := parts[0], parts[1]

		parts = strings.SplitN(ownername, ".", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", name)
		}
		uname, gname := parts[0], parts[1]

		meta.unameMap[filename] = uname
		meta.gnameMap[filename] = gname
	}

	if owner != "" {
		parts := strings.SplitN(owner, ".", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", owner)
		}
		uid, err := strconv.Atoi(parts[0])
		if err != nil {
			glog.Fatalf("could not parse uid: %q", parts[0])
		}
		gid, err := strconv.Atoi(parts[1])
		if err != nil {
			glog.Fatalf("could not parse gid: %q", parts[1])
		}
		meta.defaultUID = uid
		meta.defaultGID = gid

	}

	meta.uidMap = map[string]int{}
	meta.gidMap = map[string]int{}
	for _, owner := range owners {
		parts := strings.SplitN(owner, "=", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", owner)
		}
		filename, owner := parts[0], parts[1]

		parts = strings.SplitN(parts[1], ".", 2)
		if len(parts) != 2 {
			glog.Fatalf("expected two parts to %q", owner)
		}
		uid, err := strconv.Atoi(parts[0])
		if err != nil {
			glog.Fatalf("could not parse uid: %q", parts[0])
		}
		gid, err := strconv.Atoi(parts[1])
		if err != nil {
			glog.Fatalf("could not parse gid: %q", parts[1])
		}
		meta.uidMap[filename] = uid
		meta.gidMap[filename] = gid
	}

	return meta
}

type fileMeta struct {
	defaultGID, defaultUID int
	gidMap, uidMap         map[string]int

	defaultGname, defaultUname string
	gnameMap, unameMap         map[string]string

	defaultMode os.FileMode
	modeMap     map[string]os.FileMode
}

func (f *fileMeta) getGID(fname string) int {
	if id, ok := f.gidMap[fname]; ok {
		return id
	}
	return f.defaultGID
}

func (f *fileMeta) getUID(fname string) int {
	if id, ok := f.uidMap[fname]; ok {
		return id
	}
	return f.defaultUID
}

func (f *fileMeta) getGname(fname string) string {
	if name, ok := f.gnameMap[fname]; ok {
		return name
	}
	return f.defaultGname
}

func (f *fileMeta) getUname(fname string) string {
	if name, ok := f.unameMap[fname]; ok {
		return name
	}
	return f.defaultUname
}

func (f *fileMeta) getMode(fname string) os.FileMode {
	if mode, ok := f.modeMap[fname]; ok {
		return mode
	}
	return f.defaultMode
}

type multiString []string

func (ms *multiString) String() string {
	return strings.Join(*ms, ",")
}

func (ms *multiString) Set(v string) error {
	*ms = append(*ms, v)
	return nil
}
