package docker

import (
	"archive/tar"
	"bytes"
	"dockflow/internal/service/filesystem"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/docker/docker/api/types"
	"github.com/otiai10/copy"
	"github.com/samber/lo"
)

var BuildTypeEnum = []string{"go", "java", "node-page", "node-service", "php", "python"}

var (
	ErrorBuildTypeNotExist = errors.New("build type not exist")
	ErrorBuildPathNotExist = errors.New("build path not exist")
)

func TarBuildContext(dir string) (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return buf, nil
}

func Build(path string, tag string, buildtype string, args map[string]*string) error {
	if !lo.Contains(BuildTypeEnum, buildtype) {
		return ErrorBuildTypeNotExist
	}

	isExist, err := filesystem.DirExists(path)
	if err != nil {
		return err
	}

	if !isExist {
		return ErrorBuildPathNotExist
	}

	err = copy.Copy(filesystem.BuildDockerfilePath+buildtype, path+"/Dockerfile."+buildtype)
	if err != nil {
		print("copy dockerfile err: \n")
		return err
	}

	tarReader, err := TarBuildContext(path)
	if err != nil {
		return err
	}

	opts := types.ImageBuildOptions{
		Tags:       []string{tag},
		Dockerfile: "Dockerfile." + buildtype,
		Remove:     true,
		BuildArgs:  args,
	}

	resp, err := Client().ImageBuild(Ctx(), tarReader, opts)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)
	for {
		var msg map[string]interface{}
		if err := decoder.Decode(&msg); err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		if v, ok := msg["stream"]; ok {
			fmt.Print(v)
		}
		if v, ok := msg["error"]; ok {
			return fmt.Errorf("%v", v)
		}
	}

	return nil
}
