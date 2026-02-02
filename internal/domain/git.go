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
		return info, nil
	}

	// 处理 HTTP/HTTPS 格式的 URL
	parsedURL, err := url.Parse(gitURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 设置 host
	info.Host = parsedURL.Hostname()

	// 从 User 信息中获取 username
	if parsedURL.User != nil {
		info.Username = parsedURL.User.Username()
	}

	// 如果 URL 中包含用户名（如 https://username@host/path）
	if info.Username == "" && strings.Contains(gitURL, "@") {
		// 提取 username@host 格式的用户名
		atIndex := strings.Index(gitURL, "@")
		if atIndex > 0 {
			// 查找 username 的起始位置（在 :// 之后）
			schemeEnd := strings.Index(gitURL, "://")
			if schemeEnd != -1 && atIndex > schemeEnd+3 {
				info.Username = gitURL[schemeEnd+3 : atIndex]
			}
		}
	}

	return info, nil
}
