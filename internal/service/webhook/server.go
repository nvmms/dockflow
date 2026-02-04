package webhook

import (
	"context"
	"log"
	"net/http"
)

type Server struct {
	httpServer *http.Server
}

func NewServer(addr string, gitService *GitService) *Server {
	mux := http.NewServeMux()

	// health
	mux.HandleFunc("/webhook/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	// git webhook
	mux.HandleFunc("/webhook/git/", gitService.HandleWebhook)

	return &Server{
		httpServer: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		log.Println("[webhook] listening on", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("[webhook] error:", err)
		}
	}()

	go func() {
		<-ctx.Done()
		log.Println("[webhook] shutting down")
		_ = s.httpServer.Shutdown(context.Background())
	}()
}
