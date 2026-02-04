package git

import (
	"bytes"
	"context"
	"dockflow/internal/service/docker"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

/* ---------- Types ---------- */

type Provider string

const (
	ProviderGitHub Provider = "github"
	ProviderGitLab Provider = "gitlab"
	ProviderGitee  Provider = "gitee"
)

type WebhookOption struct {
	Provider    Provider
	Repo        string // owner/repo | gitlab project path (url-encoded)
	CallbackURL string // https://xxx/webhook/git/xxx
	Secret      string
	Token       string
	Events      []string
}

type webhook struct {
	ID  string
	URL string
}

/* ---------- Public Entry ---------- */

// EnsureWebhook 是唯一对外暴露的方法（幂等）
// ⚠️ 调用前请确保已 NormalizeWebhookOption

/* ---------- Normalize ---------- */

// NormalizeWebhookOption
// - 从 repo URL 自动识别 Provider
// - 自动提取 Repo（owner/repo 或 gitlab url-encoded path）
func NormalizeWebhookOption(opt WebhookOption) error {
	opt, err := parseFromRepoURL(opt)
	if err != nil {
		return err
	}
	switch opt.Provider {
	case ProviderGitHub:
		return ensureGitHubWebhook(docker.Ctx(), opt)
	case ProviderGitLab:
		return ensureGitLabWebhook(docker.Ctx(), opt)
	case ProviderGitee:
		return ensureGiteeWebhook(docker.Ctx(), opt)
	default:
		return errors.New("unsupported git provider")
	}
}

func parseFromRepoURL(opt WebhookOption) (WebhookOption, error) {
	u, err := url.Parse(opt.Repo)
	if err != nil {
		return opt, err
	}

	host := u.Host
	path := strings.TrimSuffix(strings.TrimPrefix(u.Path, "/"), ".git")

	switch {
	case strings.Contains(host, "gitee.com"):
		opt.Provider = ProviderGitee
		opt.Repo = path // owner/repo

	case strings.Contains(host, "github.com"):
		opt.Provider = ProviderGitHub
		opt.Repo = path // owner/repo

	default:
		// GitLab（含自建）
		opt.Provider = ProviderGitLab
		// GitLab API 支持 url-encoded path 作为 project id
		opt.Repo = url.PathEscape(path)
	}

	return opt, nil
}

/* ============================================================
   GitHub
   ============================================================ */

func ensureGitHubWebhook(ctx context.Context, opt WebhookOption) error {
	hooks, err := githubListWebhooks(ctx, opt)
	if err != nil {
		return err
	}

	for _, h := range hooks {
		if h.URL == opt.CallbackURL {
			return nil
		}
	}
	return githubCreateWebhook(ctx, opt)
}

func githubListWebhooks(ctx context.Context, opt WebhookOption) ([]webhook, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("https://api.github.com/repos/%s/hooks", opt.Repo),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "token "+opt.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []struct {
		ID     int `json:"id"`
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var hooks []webhook
	for _, r := range raw {
		hooks = append(hooks, webhook{
			ID:  fmt.Sprintf("%d", r.ID),
			URL: r.Config.URL,
		})
	}
	return hooks, nil
}

func githubCreateWebhook(ctx context.Context, opt WebhookOption) error {
	payload := map[string]any{
		"name":   "web",
		"active": true,
		"events": opt.Events,
		"config": map[string]string{
			"url":          opt.CallbackURL,
			"content_type": "json",
			"secret":       opt.Secret,
		},
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("https://api.github.com/repos/%s/hooks", opt.Repo),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "token "+opt.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("github create webhook failed: %s", resp.Status)
	}
	return nil
}

/* ============================================================
   GitLab
   ============================================================ */

func ensureGitLabWebhook(ctx context.Context, opt WebhookOption) error {
	hooks, err := gitlabListWebhooks(ctx, opt)
	if err != nil {
		return err
	}

	for _, h := range hooks {
		if h.URL == opt.CallbackURL {
			return nil
		}
	}
	return gitlabCreateWebhook(ctx, opt)
}

func gitlabListWebhooks(ctx context.Context, opt WebhookOption) ([]webhook, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/hooks", opt.Repo),
		nil,
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Private-Token", opt.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	var hooks []webhook
	for _, r := range raw {
		hooks = append(hooks, webhook{
			ID:  fmt.Sprintf("%d", r.ID),
			URL: r.URL,
		})
	}
	return hooks, nil
}

func gitlabCreateWebhook(ctx context.Context, opt WebhookOption) error {
	payload := map[string]any{
		"url":                   opt.CallbackURL,
		"token":                 opt.Secret,
		"push_events":           true,
		"tag_push_events":       true,
		"merge_requests_events": true,
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("https://gitlab.com/api/v4/projects/%s/hooks", opt.Repo),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Private-Token", opt.Token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitlab create webhook failed: %s", resp.Status)
	}
	return nil
}

/* ============================================================
   Gitee
   ============================================================ */

func ensureGiteeWebhook(ctx context.Context, opt WebhookOption) error {
	owner, repo, err := splitOwnerRepo(opt.Repo)
	if err != nil {
		return err
	}

	hooks, err := giteeListWebhooks(ctx, owner, repo, opt.Token)
	if err != nil {
		return err
	}

	for _, h := range hooks {
		if h.URL == opt.CallbackURL {
			return nil
		}
	}
	return giteeCreateWebhook(ctx, owner, repo, opt)
}

func giteeListWebhooks(ctx context.Context, owner, repo, token string) ([]webhook, error) {
	url := fmt.Sprintf(
		"https://gitee.com/api/v5/repos/%s/%s/hooks?access_token=%s",
		owner, repo, token,
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("gitee list webhooks failed: %s", resp.Status)
	}

	var raw []struct {
		ID  int    `json:"id"`
		URL string `json:"url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}

	hooks := make([]webhook, 0, len(raw))
	for _, r := range raw {
		hooks = append(hooks, webhook{
			ID:  fmt.Sprintf("%d", r.ID),
			URL: r.URL,
		})
	}
	return hooks, nil
}

func giteeCreateWebhook(ctx context.Context, owner, repo string, opt WebhookOption) error {
	payload := map[string]any{
		"access_token":    opt.Token,
		"url":             opt.CallbackURL,
		"password":        opt.Secret,
		"push_events":     true,
		"tag_push_events": true,
	}

	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("https://gitee.com/api/v5/repos/%s/%s/hooks", owner, repo),
		bytes.NewReader(body),
	)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json;charset=UTF-8")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("gitee create webhook failed: %s", resp.Status)
	}
	return nil
}

func splitOwnerRepo(full string) (string, string, error) {
	parts := strings.Split(full, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repo format: %s (want owner/repo)", full)
	}
	return parts[0], parts[1], nil
}

/* ---------- Helper ---------- */

func defaultTimeoutCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
