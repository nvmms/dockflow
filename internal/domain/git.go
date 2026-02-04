package domain

import (
	"fmt"
	"net/url"
	"strings"
)

type GitURLInfo struct {
	Host     string
	Username string
	URL      string
	Path     string
	Repo     string
}

// ParseGitURL 解析 Git URL，支持多种格式
func NewGitUrl(gitURL string) (*GitURLInfo, error) {
	info := &GitURLInfo{URL: gitURL}

	// 处理 SSH 格式的 URL（git@github.com:user/repo.git）
	if strings.HasPrefix(gitURL, "git@") {
		parts := strings.SplitN(gitURL, "@", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid SSH git URL format")
		}

		info.Username = "git"

		hostPath := parts[1]
		colonIndex := strings.Index(hostPath, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("invalid SSH git URL format - missing colon")
		}

		info.Host = hostPath[:colonIndex]
		info.Path = hostPath[colonIndex+1:]

		// 为 SSH 格式提取 username 和 repo
		extractUsernameAndRepo(info)
		return info, nil
	}

	// 处理 HTTP/HTTPS 格式的 URL
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	info.Host = parsedURL.Hostname()
	info.Path = strings.TrimPrefix(parsedURL.Path, "/")

	// 从 User 信息中获取 username
	if parsedURL.User != nil {
		info.Username = parsedURL.User.Username()
	}

	// 提取 username 和 repo
	extractUsernameAndRepo(info)

	return info, nil
}

// 提取 username 和 repo
func extractUsernameAndRepo(info *GitURLInfo) {
	cleanPath := strings.TrimSuffix(info.Path, ".git")
	cleanPath = strings.TrimSuffix(cleanPath, "/")
	pathParts := strings.Split(cleanPath, "/")

	if len(pathParts) >= 1 && pathParts[0] != "" {
		// 如果 username 还没设置，设置第一个部分为 username
		if info.Username == "" {
			info.Username = pathParts[0]
		}

		// 提取 repo（第二个部分）
		if len(pathParts) >= 2 {
			info.Repo = pathParts[1]
		} else if len(pathParts) == 1 {
			// 如果只有一个部分，可能是没有 username 的 repo
			info.Repo = pathParts[0]
		}
	}
}
