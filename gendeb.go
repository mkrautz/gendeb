package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/md5"
	"encoding/hex"
	"flag"
	"github.com/mkrautz/goar"
	"io/ioutil"
	"json"
	"log"
	"os"
	"strconv"
)

type Spec struct {
	Control map[string]string
	Files   []File

	md5sums *bytes.Buffer
	data    *bytes.Buffer
	control *bytes.Buffer
}

type File struct {
	Name string
	Dest string
	Mode string
	Uid  int
	Gid  int
}

func NewSpec(filename string) (*Spec, os.Error) {
	spec := new(Spec)
	spec.md5sums = new(bytes.Buffer)
	spec.data = new(bytes.Buffer)
	spec.control = new(bytes.Buffer)

	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	if err != nil {
		return nil, err
	}
	err = dec.Decode(spec)
	if err != nil {
		return nil, err
	}

	if Args.Version != "" {
		spec.Control["Version"] = Args.Version
	}

	if _, exists := spec.Control["Package"]; !exists {
		return nil, os.NewError("spec: Missing required control key: Package")
	}
	if _, exists := spec.Control["Version"]; !exists {
		return nil, os.NewError("spec: Missing required control key: Version")
	}
	if _, exists := spec.Control["Architecture"]; !exists {
		return nil, os.NewError("spec: Missing required control key: Architecture")
	}
	return spec, nil
}

func (spec *Spec) Filename() string {
	if Args.Out != "" {
		return Args.Out
	}
	return spec.Control["Package"] + "_" + spec.Control["Version"] + "_" + spec.Control["Architecture"] + ".deb"
}

func (spec *Spec) controlFile() ([]byte, os.Error) {
	buf := new(bytes.Buffer)
	for k, v := range spec.Control {
		_, err := buf.WriteString(k)
		if err != nil {
			return nil, err
		}
		_, err = buf.WriteString(": ")
		if err != nil {
			return nil, err
		}
		_, err = buf.WriteString(v)
		if err != nil {
			return nil, err
		}
		_, err = buf.WriteString("\n")
		if err != nil {
			return nil, err
		}
	}
	return buf.Bytes(), nil
}

func (spec *Spec) GenerateDeb() os.Error {
	czw, err := gzip.NewWriter(spec.control)
	if err != nil {
		return err
	}
	controlWriter := tar.NewWriter(czw)

	// Write control file
	controlFile, err := spec.controlFile()
	if err != nil {
		return err
	}
	err = controlWriter.WriteHeader(&tar.Header{
		Name: "./control",
		Size: int64(len(controlFile)),
		Mode: int64(0644),
	})
	if err != nil {
		return err
	}
	_, err = controlWriter.Write(controlFile)
	if err != nil {
		return err
	}

	// Write all files
	dzw, err := gzip.NewWriter(spec.data)
	if err != nil {
		return err
	}
	dataWriter := tar.NewWriter(dzw)
	for _, file := range spec.Files {
		buf, err := ioutil.ReadFile(file.Name)
		if err != nil {
			return err
		}

		hasher := md5.New()
		_, err = hasher.Write(buf)
		if err != nil {
			return err
		}
		md5sum := hex.EncodeToString(hasher.Sum())

		mode, err := strconv.Btoi64(file.Mode, 8)
		err = dataWriter.WriteHeader(&tar.Header{
			Name: file.Dest,
			Mode: mode,
			Uid:  file.Uid,
			Gid:  file.Gid,
			Size: int64(len(buf)),
		})
		if err != nil {
			return err
		}

		_, err = dataWriter.Write(buf)
		if err != nil {
			return err
		}

		_, err = spec.md5sums.WriteString(md5sum)
		if err != nil {
			return err
		}
		_, err = spec.md5sums.WriteString("  ")
		if err != nil {
			return err
		}
		_, err = spec.md5sums.WriteString(file.Dest)
		if err != nil {
			return err
		}
		_, err = spec.md5sums.WriteString("\n")
		if err != nil {
			return err
		}
	}

	// Write md5sums to control.tar.gz
	err = controlWriter.WriteHeader(&tar.Header{
		Name: "./md5sums",
		Size: int64(spec.md5sums.Len()),
		Mode: int64(0644),
	})
	if err != nil {
		return err
	}

	_, err = controlWriter.Write(spec.md5sums.Bytes())
	if err != nil {
		return err
	}

	// Close all writers in preparation of writing to the .deb
	err = controlWriter.Close()
	if err != nil {
		return err
	}

	err = czw.Close()
	if err != nil {
		return err
	}

	err = dataWriter.Close()
	if err != nil {
		return err
	}

	err = dzw.Close()
	if err != nil {
		return err
	}

	// Write files to the deb
	debFile, err := os.Create(spec.Filename())
	if err != nil {
		return err
	}

	aw := ar.NewWriter(debFile)
	err = aw.WriteHeader(&ar.Header{
		Name: "debian-binary",
		Size: 4,
		Mode: int64(0644),
	})
	if err != nil {
		return err
	}

	_, err = aw.Write([]byte{'2', '.', '0', '\n'})
	if err != nil {
		return err
	}

	err = aw.WriteHeader(&ar.Header{
		Name: "control.tar.gz",
		Size: int64(spec.control.Len()),
		Mode: int64(0644),
	})
	_, err = aw.Write(spec.control.Bytes())
	if err != nil {
		return err
	}

	err = aw.WriteHeader(&ar.Header{
		Name: "data.tar.gz",
		Size: int64(spec.data.Len()),
		Mode: int64(0644),
	})
	_, err = aw.Write(spec.data.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if Args.Spec == "" || Args.ShowHelp == true {
		Usage()
		return
	}

	spec, err := NewSpec(Args.Spec)
	if err != nil {
		log.Fatalf("Couldn't read spec file: %v", err)
	}

	err = spec.GenerateDeb()
	if err != nil {
		log.Fatalf("Unable to generate deb: %v", err)
	}

	log.Printf("Wrote deb file to: %v", spec.Filename())
}
