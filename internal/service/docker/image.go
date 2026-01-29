package docker

import (
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/moby/term"
)

func EnsureImage(image string) error {
	if imageExists(image) {
		return nil
	}

	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func imageExists(image string) bool {
	cmd := exec.Command("docker", "image", "inspect", image)
	return cmd.Run() == nil
}

func PullImage(name string) error {
	isExists, err := ImageExists(name)
	if err != nil {
		return err
	}
	if isExists {
		return nil
	}
	ctx := Ctx()

	rc, err := Client().ImagePull(ctx, name, types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer rc.Close()

	// 关键：让输出像 docker pull 一样（TTY 刷新 + 进度条）
	fd, isTerm := term.GetFdInfo(os.Stdout)

	return jsonmessage.DisplayJSONMessagesStream(
		rc,
		os.Stdout,
		fd,
		isTerm,
		nil,
	)
}

func ImageExists(name string) (bool, error) {
	_, _, err := Client().ImageInspectWithRaw(Ctx(), name)
	if err == nil {
		return true, nil
	}

	// 镜像不存在
	if client.IsErrNotFound(err) {
		return false, nil
	}

	// 其他错误（Docker daemon 异常等）
	return false, err
}
