package webhook

import (
	"encoding/json"
	"strings"
)

const (
	GitRefBranch GitRefType = "branch"
	GitRefTag    GitRefType = "tag"
)

// ---------- GitHub ----------
func (s *GitService) handleGitHub(ns string, appName string, body []byte) {
	var p struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		HeadCommit struct {
			ID string `json:"id"`
		} `json:"head_commit"`
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return
	}

	refType, refName := parseGitRef(p.Ref)

	event := GitPushEvent{
		Namespace: ns,
		AppName:   appName,
		Provider:  "github",

		Repo:    p.Repository.FullName,
		Ref:     p.Ref,
		RefType: refType,
		RefName: refName,

		Commit: p.HeadCommit.ID,
	}

	s.Handle(event)
}

// ---------- GitLab ----------
func (s *GitService) handleGitLab(ns string, appName string, body []byte) {
	var p struct {
		Ref     string `json:"ref"`
		Project struct {
			Path string `json:"path_with_namespace"`
		} `json:"project"`
		CheckoutSha string `json:"checkout_sha"`
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return
	}

	refType, refName := parseGitRef(p.Ref)

	event := GitPushEvent{
		Namespace: ns,
		AppName:   appName,
		Provider:  "github",

		Repo:    p.Project.Path,
		Ref:     p.Ref,
		RefType: refType,
		RefName: refName,

		Commit: p.CheckoutSha,
	}

	s.Handle(event)
}

// ---------- Gitee ----------
func (s *GitService) handleGitee(ns string, appName string, body []byte) {
	var p struct {
		Ref        string `json:"ref"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
		HeadCommit struct {
			ID string `json:"id"`
		} `json:"head_commit"`
	}

	if err := json.Unmarshal(body, &p); err != nil {
		return
	}

	refType, refName := parseGitRef(p.Ref)

	event := GitPushEvent{
		Namespace: ns,
		AppName:   appName,
		Provider:  "gitee",

		Repo:    p.Repository.FullName,
		Ref:     p.Ref,
		RefType: refType,
		RefName: refName,

		Commit: p.HeadCommit.ID,
	}

	s.Handle(event)
}

// ---------- util ----------

func parseBranch(ref string) string {
	const prefix = "refs/heads/"
	if len(ref) > len(prefix) && ref[:len(prefix)] == prefix {
		return ref[len(prefix):]
	}
	return ref
}

func parseGitRef(ref string) (GitRefType, string) {
	const (
		branchPrefix = "refs/heads/"
		tagPrefix    = "refs/tags/"
	)

	switch {
	case strings.HasPrefix(ref, branchPrefix):
		return GitRefBranch, ref[len(branchPrefix):]

	case strings.HasPrefix(ref, tagPrefix):
		return GitRefTag, ref[len(tagPrefix):]

	default:
		return "", ""
	}
}
