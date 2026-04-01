package server
import("encoding/json";"net/http";"github.com/stockyard-dev/stockyard-billfold/internal/store")
type Server struct{db *store.DB;limits Limits;mux *http.ServeMux}
func New(db *store.DB,tier string)*Server{s:=&Server{db:db,limits:LimitsFor(tier),mux:http.NewServeMux()};s.routes();return s}
func(s *Server)ListenAndServe(addr string)error{return(&http.Server{Addr:addr,Handler:s.mux}).ListenAndServe()}
func(s *Server)routes(){
    s.mux.HandleFunc("GET /health",s.handleHealth)
    s.mux.HandleFunc("GET /api/stats",s.handleStats)
    s.mux.HandleFunc("GET /api/clients",s.handleListClients)
    s.mux.HandleFunc("POST /api/clients",s.handleCreateClient)
    s.mux.HandleFunc("DELETE /api/clients/{id}",s.handleDeleteClient)
    s.mux.HandleFunc("GET /api/invoices",s.handleListInvoices)
    s.mux.HandleFunc("POST /api/invoices",s.handleCreateInvoice)
    s.mux.HandleFunc("PATCH /api/invoices/{id}",s.handleUpdateInvoice)
    s.mux.HandleFunc("DELETE /api/invoices/{id}",s.handleDeleteInvoice)
    s.mux.HandleFunc("GET /api/invoices/{id}/items",s.handleGetLineItems)
    s.mux.HandleFunc("POST /api/invoices/{id}/items",s.handleAddLineItem)
    s.mux.HandleFunc("DELETE /api/items/{id}",s.handleDeleteLineItem)
    s.mux.HandleFunc("GET /",s.handleUI)
}
func(s *Server)handleHealth(w http.ResponseWriter,r *http.Request){writeJSON(w,200,map[string]string{"status":"ok","service":"stockyard-billfold"})}
func writeJSON(w http.ResponseWriter,status int,v interface{}){w.Header().Set("Content-Type","application/json");w.WriteHeader(status);json.NewEncoder(w).Encode(v)}
func writeError(w http.ResponseWriter,status int,msg string){writeJSON(w,status,map[string]string{"error":msg})}
func(s *Server)handleUI(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};w.Header().Set("Content-Type","text/html");w.Write(dashboardHTML)}
