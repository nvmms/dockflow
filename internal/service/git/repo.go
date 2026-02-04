package git

import (
	dockflowConfig "dockflow/internal/config"
	"dockflow/internal/domain"
	"dockflow/internal/service/filesystem"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage/memory"
)

type GitCloneOptions struct {
	RepoURL string
	DestDir string

	Branch *string // 分支名（仅用于解析 commit）
	Commit *string // commit hash（最高优先级）
	Tag    *string // tag 名（用于解析 commit）

	Token string
}

var (
	ErrorCommitHashBlank   = errors.New("commit hash can't be blank")
	ErrorRepoDestRequired  = errors.New("repo url and dest dir required")
	ErrorResolveCommitFail = errors.New("failed to resolve commit")
)

/*
ResolveCommit
职责：将 branch / tag / commit 统一解析为最终 commit hash
优先级：commit > tag > branch
*/
func ResolveCommit(opts GitCloneOptions) (string, error) {
	// 1) commit 最高优先级
	if opts.Commit != nil && *opts.Commit != "" {
		return *opts.Commit, nil
	}

	if opts.RepoURL == "" {
		return "", ErrorRepoDestRequired
	}

	remote := git.NewRemote(memory.NewStorage(), &config.RemoteConfig{
		Name: "origin",
		URLs: []string{opts.RepoURL},
	})

	listOpts := &git.ListOptions{}
	listOpts.Auth = auth(opts)

	refs, err := remote.List(listOpts)

	if err != nil {
		panic(err)
	}

	// 小工具：按 refname 查 hash
	findHash := func(name plumbing.ReferenceName) (string, bool) {
		for _, r := range refs {
			if r.Name() == name {
				return r.Hash().String(), true
			}
		}
		return "", false
	}

	// 2) tag
	if opts.Tag != nil && *opts.Tag != "" {
		// 注意：轻量 tag 是 refs/tags/x -> hash
		// 注：annotated tag 这里拿到的是 tag object hash，不是 commit
		// 如需严格解析 annotated tag -> commit，需要再展开 tag object（见下方备注）
		refName := plumbing.NewTagReferenceName(*opts.Tag)
		if h, ok := findHash(refName); ok {
			return h, nil
		}
		return "", fmt.Errorf("tag not found: %s", *opts.Tag)
	}

	// 3) branch
	if opts.Branch != nil && *opts.Branch != "" {
		refName := plumbing.NewBranchReferenceName(*opts.Branch)
		if h, ok := findHash(refName); ok {
			return h, nil
		}
		return "", fmt.Errorf("branch not found: %s", *opts.Branch)
	}

	// 4) 都没传：取默认分支最新 commit
	//
	// 优先解析 origin/HEAD（symbolic ref）指向的分支
	// 常见形式：
	// - refs/remotes/origin/HEAD -> refs/remotes/origin/main（符号引用）
	// - HEAD -> refs/heads/main（某些服务）
	//
	// go-git 的 remote.List 返回的 *plumbing.Reference 可能是 HashReference 或 SymbolicReference
	// 我们先找 Symbolic 的 HEAD，再根据 Target 找对应 heads hash
	var headTargets []plumbing.ReferenceName
	for _, r := range refs {
		if r.Name() == plumbing.ReferenceName("refs/remotes/origin/HEAD") || r.Name() == plumbing.HEAD {
			if r.Type() == plumbing.SymbolicReference {
				headTargets = append(headTargets, r.Target())
			}
		}
	}

	// 把 target 规范成 refs/heads/*
	normalizeToHeads := func(target plumbing.ReferenceName) plumbing.ReferenceName {
		// target 可能是 refs/remotes/origin/main 或 refs/heads/main
		s := target.String()
		if strings.HasPrefix(s, "refs/remotes/origin/") {
			return plumbing.ReferenceName("refs/heads/" + strings.TrimPrefix(s, "refs/remotes/origin/"))
		}
		return target
	}

	for _, t := range headTargets {
		headBranch := normalizeToHeads(t)
		if h, ok := findHash(headBranch); ok {
			return h, nil
		}
	}

	// fallback：优先 main/master
	if h, ok := findHash(plumbing.NewBranchReferenceName("main")); ok {
		return h, nil
	}
	if h, ok := findHash(plumbing.NewBranchReferenceName("master")); ok {
		return h, nil
	}

	// 再兜底：取任意一个 refs/heads/*
	for _, r := range refs {
		if r.Name().IsBranch() {
			return r.Hash().String(), nil
		}
	}

	return "", ErrorResolveCommitFail
}

/*
EnsureRepo
职责：
- 本地不存在 → clone
- 存在但非 git repo → 删除后 clone
- 已存在 repo → fetch 更新 refs
*/
func EnsureRepo(opts GitCloneOptions) (*git.Repository, error) {
	if opts.RepoURL == "" || opts.DestDir == "" {
		return nil, ErrorRepoDestRequired
	}

	exist, err := filesystem.DirExists(opts.DestDir)
	if err != nil {
		return nil, err
	}

	// 不存在目录 → clone
	if !exist {
		return cloneRepo(opts)
	}

	// 存在但不是 git 仓库 → 删除重建
	if ok, _ := filesystem.DirExists(opts.DestDir + "/.git"); !ok {
		if err := os.RemoveAll(opts.DestDir); err != nil {
			return nil, err
		}
		return cloneRepo(opts)
	}

	repo, err := git.PlainOpen(opts.DestDir)
	if err != nil {
		return nil, err
	}

	// fetch 最新 refs（不 merge）
	err = repo.Fetch(&git.FetchOptions{
		RemoteName: "origin",
		Auth:       auth(opts),
		Force:      true,
		Tags:       git.AllTags,
	})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return nil, err
	}

	return repo, nil
}

/*
CheckoutCommit
职责：checkout 到指定 commit
*/
func CheckoutCommit(repo *git.Repository, commit string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return err
	}

	println(repo)
	println(commit)

	return wt.Checkout(&git.CheckoutOptions{
		Hash:  plumbing.NewHash(commit),
		Force: true,
	})
}

/*
GetLatestCode
统一入口：
1. 解析最终 commit
2. 确保仓库存在
3. checkout 到 commit
4. 返回 version（short commit）
*/
func GetLatestCode(opts GitCloneOptions) (string, error) {
	commit, err := ResolveCommit(opts)
	if err != nil {
		return "", err
	}

	repo, err := EnsureRepo(opts)
	if err != nil {
		return "", err
	}

	if err := CheckoutCommit(repo, commit); err != nil {
		return "", err
	}

	if len(commit) > 7 {
		return commit[:7], nil
	}
	return commit, nil
}

/* ---------- internal helpers ---------- */

func cloneRepo(opts GitCloneOptions) (*git.Repository, error) {
	cloneOpts := &git.CloneOptions{
		URL: opts.RepoURL,
	}

	cloneOpts.Auth = auth(opts)

	return git.PlainClone(opts.DestDir, false, cloneOpts)
}

func auth(opts GitCloneOptions) transport.AuthMethod {
	if opts.Token != "" {
		return &http.BasicAuth{
			Username: "oauth2",
			Password: opts.Token,
		}
	} else {
		gitInfo, err := domain.NewGitUrl(opts.RepoURL)
		if err != nil {
			panic(err)
		}

		token, err := dockflowConfig.FindGit(gitInfo.Host, gitInfo.Username)
		if err != nil {
			panic(err)
		}
		return &http.BasicAuth{
			Username: "oauth2",
			Password: token,
		}
	}
}
