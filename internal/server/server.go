package server

import (
	"encoding/json"
	"net/http"

	"github.com/stockyard-dev/stockyard-billfold/internal/store"
)

type Server struct {
	db     *store.DB
	limits Limits
	mux    *http.ServeMux
}

func New(db *store.DB, tier string) *Server {
	s := &Server{
		db:     db,
		limits: LimitsFor(tier),
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{Addr: addr, Handler: s.mux}
	return srv.ListenAndServe()
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/version", s.handleVersion)
	s.mux.HandleFunc("GET /api/limits", s.handleLimits)
	s.mux.HandleFunc("GET /", s.handleUI)
	s.mux.HandleFunc("GET /api/items", s.handleListItems)
	s.mux.HandleFunc("POST /api/items", s.handleCreateItem)
	s.mux.HandleFunc("GET /api/items/{id}", s.handleGetItem)
	s.mux.HandleFunc("PUT /api/items/{id}", s.handleUpdateItem)
	s.mux.HandleFunc("DELETE /api/items/{id}", s.handleDeleteItem)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "stockyard-billfold"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"version": "0.1.0", "service": "stockyard-billfold"})
}

func (s *Server) handleLimits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tier":        s.limits.Tier,
		"description": s.limits.Description,
		"is_pro":      s.limits.IsPro(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
