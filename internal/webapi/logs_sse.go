package webapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"dockflow/internal/usecase"

	"github.com/docker/docker/pkg/stdcopy"
)

type logEvent struct {
	Chunk  string `json:"chunk,omitempty"`
	Status string `json:"status,omitempty"`
}

func validateLogTail(r *http.Request) (string, error) {
	tail := r.URL.Query().Get("tail")
	if tail == "" {
		return "200", nil
	}
	value, err := strconv.Atoi(tail)
	if err != nil || value < 1 || value > 5000 {
		return "", fmt.Errorf("tail must be between 1 and 5000")
	}
	return tail, nil
}

func prepareSSE(w http.ResponseWriter) (http.Flusher, bool) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming is unsupported"})
		return nil, false
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	return flusher, true
}

func sendLogEvent(w io.Writer, flusher http.Flusher, event string, payload logEvent) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if _, err = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func streamDeploymentLogs(w http.ResponseWriter, r *http.Request, namespace, appName, jobID string) {
	job, err := usecase.GetDeploymentJob(namespace, appName, jobID)
	if err != nil {
		writeError(w, err)
		return
	}
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}
	offset := 0
	ticker := time.NewTicker(350 * time.Millisecond)
	defer ticker.Stop()
	for {
		job, err = usecase.GetDeploymentJob(namespace, appName, jobID)
		if err != nil {
			_ = sendLogEvent(w, flusher, "error", logEvent{Status: err.Error()})
			return
		}
		if offset < len(job.Logs) {
			if err := sendLogEvent(w, flusher, "log", logEvent{Chunk: job.Logs[offset:]}); err != nil {
				return
			}
			offset = len(job.Logs)
		}
		if job.Status != "running" {
			_ = sendLogEvent(w, flusher, "done", logEvent{Status: job.Status})
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
		}
	}
}

type sseLogWriter struct {
	w       io.Writer
	flusher http.Flusher
}

func (writer sseLogWriter) Write(data []byte) (int, error) {
	if err := sendLogEvent(writer.w, writer.flusher, "log", logEvent{Chunk: string(data)}); err != nil {
		return 0, err
	}
	return len(data), nil
}

func streamContainerLogs(w http.ResponseWriter, r *http.Request, open func(string) (io.ReadCloser, error)) {
	tail, err := validateLogTail(r)
	if err != nil {
		writeBadRequest(w, err.Error())
		return
	}
	stream, err := open(tail)
	if err != nil {
		writeError(w, err)
		return
	}
	defer stream.Close()
	flusher, ok := prepareSSE(w)
	if !ok {
		return
	}
	go func() { <-r.Context().Done(); _ = stream.Close() }()
	writer := sseLogWriter{w: w, flusher: flusher}
	_, err = stdcopy.StdCopy(writer, writer, stream)
	if err == nil && r.Context().Err() == nil {
		_ = sendLogEvent(w, flusher, "done", logEvent{Status: "closed"})
	}
}
