package docker

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
)

func BuildWithNpmService(path string, tag string) {
	// TODO: 打包npm程序，自己提供程序访问，
}
func BuildWithNpmWeb(path string, tag string) {
	// TODO: 打包npm 前端应用，需要配合nginx\apache，提供访问
}

func BuildWithJava(path string, tag string) {}

func BuildWithPhp(path string, tag string) {}

func BuildWithPython(path string, tag string) {}

func BuildWithGo(path string, tag string) {}

func tarDirectory(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(dir, func(file string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(fi, "")
		if err != nil {
			return err
		}
		header.Name = file

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if fi.IsDir() {
			return nil
		}

		f, err := os.Open(file)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = io.Copy(tw, f)
		return err
	})

	if err != nil {
		return nil, err
	}

	tw.Close()
	return buf, nil
}
