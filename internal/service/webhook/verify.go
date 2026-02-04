package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
)

// ---------- GitHub HMAC ----------

func verifyGitHubSignature(secret string, body []byte, sig string) bool {
	if sig == "" || secret == "" {
		return false
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)

	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// ---------- GitLab / Gitee Token ----------

func verifySimpleToken(r *http.Request, expected string) bool {
	if expected == "" {
		return false
	}

	token := r.Header.Get("X-Gitlab-Token")
	if token == "" {
		token = r.Header.Get("X-Gitee-Token")
	}

	return token != "" &&
		subtle.ConstantTimeCompare([]byte(token), []byte(expected)) == 1
}
