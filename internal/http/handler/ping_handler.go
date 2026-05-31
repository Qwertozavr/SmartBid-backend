package handler

import "net/http"

type PingHandler struct{}

func NewPingHandler() *PingHandler {
	return &PingHandler{}
}

func (h *PingHandler) Ping(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
