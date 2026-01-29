package docker

import (
	"os"
	"os/exec"

	"github.com/docker/docker/api/types"
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
