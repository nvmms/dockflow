package git

import (
	"fmt"

	"github.com/go-git/go-git/v5"
)

func Clone(repo string, branch *string, tag *string, localPath string) {
	opts := &git.CloneOptions{
		URL:          repo,
		SingleBranch: true,
		Progress:     nil,
	}
	if branch == nil {
		opts.ReferenceName = ""
	}
	_, err := git.PlainClone(localPath, false, opts)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("clone success")
}
