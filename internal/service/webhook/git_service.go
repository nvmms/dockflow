package webhook

import (
	"dockflow/internal/domain"
	"dockflow/internal/usecase"
	"log"
	"path/filepath"
	"strings"
)

type GitRefType string

type GitPushEvent struct {
	Namespace string
	AppName   string
	Provider  string

	Repo    string
	Ref     string     // 原始 ref
	RefType GitRefType // branch / tag
	RefName string     // main / v1.0.0

	Commit string
}

type GitService struct {
}

func NewGitService() *GitService {
	return &GitService{}
}

func (s *GitService) matchTriggerRule(rule string, refName string) bool {
	// 约定：空规则等价于不匹配
	if rule == "" {
		return false
	}

	// * 直接匹配一切
	if rule == "*" {
		return true
	}

	ok, err := filepath.Match(rule, refName)
	if err != nil {
		return false
	}

	return ok
}

func (s *GitService) Handle(event GitPushEvent) {
	log.Println("[git push]",
		"namespace=", event.Namespace,
		"app_name=", event.AppName,
		"provider=", event.Provider,
		"repo=", event.Repo,
		"branch=", event.Ref,
		"commit=", event.Commit,
	)

	ns, err := domain.NewNamespace(event.Namespace)
	if err != nil {
		log.Println("[webhook][error]", err)
		return
	}
	if ns == nil {
		log.Printf("[webhook][error] namespace [%s] not found", event.Namespace)
		return
	}

	app, found := ns.FindApp(event.AppName)
	if !found {
		log.Printf("[webhook][error] app [%s] not found", event.AppName)
		return
	}
	if app.Trigger.Type != string(event.RefType) {
		log.Printf("[webhook][info] app [%s] update ref type is [%s],current ref type [%s]", event.AppName, app.Trigger.Type, event.RefType)
		return
	}

	if event.RefType == "branch" && app.Trigger.Rule != event.RefName {
		log.Printf("[webhook][info] app [%s] update ref type is [%s/%s],current ref type [%s/%s]", event.AppName, app.Trigger.Type, app.Trigger.Rule, event.RefType, event.RefName)
		return
	}

	if event.RefType == "tag" && !s.matchTriggerRule(app.Trigger.Rule, event.RefName) {
		log.Printf("[webhook][info] app [%s] update ref type is [%s/%s],current ref type [%s/%s]", event.AppName, app.Trigger.Type, app.Trigger.Rule, event.RefType, event.RefName)
		return
	}

	opt := usecase.DeployAppOptions{
		Namespace: event.Namespace,
		Name:      event.AppName,
	}
	switch event.RefType {
	case "branch":
		opt.Branch = strings.Replace(event.Ref, "refs/heads/", "", 1)
		opt.Commit = event.Commit
	case "tag":
		opt.Tag = event.RefName
	default:
		log.Printf("[webhook][error] err RefType [%s]", event.RefType)
		return
	}

	err = usecase.DeployApp(opt)
	if err != nil {
		panic(err)
		log.Println("[webhook][error] DeployApp error", err)
		return
	}

}
