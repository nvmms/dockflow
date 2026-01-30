package docker

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
)

func BuildWithNpm(path string, tag string) {

	buildCtx, err := tarDirectory(path)
	if err != nil {
		panic(err)
	}

	Client().ImageBuild(Ctx(), buildCtx, types.ImageBuildOptions{
		Tags:       []string{"demo:latest"},
		Dockerfile: "Dockerfile",
		Remove:     true,
	})
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
