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

		// 提取 host 和路径
		hostPath := parts[1]
		colonIndex := strings.Index(hostPath, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("invalid SSH git URL format - missing colon")
		}

		info.Host = hostPath[:colonIndex]
		info.Path = hostPath[colonIndex+1:]
		return info, nil
	}

	// 处理 HTTP/HTTPS 格式的 URL
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 设置 host
	info.Host = parsedURL.Hostname()
	info.Path = strings.TrimPrefix(parsedURL.Path, "/")

	// 从 User 信息中获取 username（如 https://token@github.com/...）
	if parsedURL.User != nil {
		info.Username = parsedURL.User.Username()
	} else {
		// 尝试从路径中提取用户名（如 https://github.com/username/repo.git）
		// 移除可能的 .git 后缀
		cleanPath := strings.TrimSuffix(info.Path, ".git")
		pathParts := strings.Split(cleanPath, "/")

		// 根据常见的 Git 托管平台规则判断
		if len(pathParts) >= 2 {
			// 格式：username/repository 或 organization/repository
			// 这里假设第一个部分是用户名或组织名
			info.Username = pathParts[0]
		} else if len(pathParts) == 1 && pathParts[0] != "" {
			// 格式：username 或 repository
			info.Username = pathParts[0]
		}
	}

	return info, nil
}
