package webhook

import (
	"dockflow/internal/service/filesystem"
	"io"
	"log"
	"net/http"
	"strings"
)

func (s *GitService) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		log.Println("[webhook][error]", "StatusMethodNotAllowed")
		return
	}

	nsName, appName := extractNsAndApp(r.URL.Path)
	if nsName == "" || appName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("missing namespace or app"))
		log.Println("[webhook][error]", "missing namespace or app")
		return
	}

	// 读取 body（后面要用于签名校验）
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Println("[webhook][error]", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	ns, err := filesystem.LoadNamespace(nsName)
	if err != nil {
		log.Println("[webhook][error]", err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if ns == nil {
		log.Printf("[webhook][error] namespace [%s] not found\n", nsName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	app, found := ns.FindApp(appName)
	if !found {
		log.Printf("[webhook][error] app [%s] not found\n", appName)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// 读取应用配置（你现有的方式）
	provider := detectGitProvider(r.Header)

	// ---------- Webhook 安全校验 ----------
	switch provider {
	case "github":
		if !verifyGitHubSignature(
			app.Secret,
			body,
			r.Header.Get("X-Hub-Signature-256"),
		) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid github signature"))
			log.Println("[webhook][error] invalid github signature")
			return
		}

	case "gitlab":
		if !verifySimpleToken(r, app.Secret) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid gitlab token"))
			log.Println("[webhook][error] invalid github signature")
			return
		}

	case "gitee":
		if !verifySimpleToken(r, app.Secret) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("invalid gitee token"))
			log.Println("[webhook][error] invalid github signature")
			return
		}

	default:
		w.WriteHeader(http.StatusBadRequest)
		log.Println("[webhook][error] error git provider")
		return
	}

	// ---------- 校验通过，进入业务 ----------
	switch provider {
	case "github":
		s.handleGitHub(nsName, appName, body)
	case "gitlab":
		s.handleGitLab(nsName, appName, body)
	case "gitee":
		s.handleGitee(nsName, appName, body)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func detectGitProvider(h http.Header) string {
	if h.Get("X-GitHub-Event") != "" {
		return "github"
	}
	if h.Get("X-Gitlab-Event") != "" {
		return "gitlab"
	}
	if h.Get("X-Gitee-Event") != "" {
		return "gitee"
	}
	return ""
}

func extractNsAndApp(path string) (ns string, app string) {
	// expected: /webhook/git/{ns}/{app}
	const prefix = "/webhook/git/"

	if len(path) <= len(prefix) {
		return "", ""
	}

	rest := path[len(prefix):] // {ns}/{app}
	parts := strings.Split(rest, "/")

	if len(parts) != 2 {
		return "", ""
	}

	return parts[0], parts[1]
}
