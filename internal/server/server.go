package server
import ("encoding/json";"log";"net/http";"github.com/stockyard-dev/stockyard-billfold/internal/store")
type Server struct{db *store.DB;mux *http.ServeMux;limits Limits}
func New(db *store.DB,limits Limits)*Server{s:=&Server{db:db,mux:http.NewServeMux(),limits:limits}
s.mux.HandleFunc("GET /api/invoices",s.list)
s.mux.HandleFunc("POST /api/invoices",s.create)
s.mux.HandleFunc("GET /api/invoices/{id}",s.get)
s.mux.HandleFunc("PUT /api/invoices/{id}",s.update)
s.mux.HandleFunc("DELETE /api/invoices/{id}",s.del)
s.mux.HandleFunc("GET /api/stats",s.stats)
s.mux.HandleFunc("GET /api/health",s.health)
s.mux.HandleFunc("GET /ui",s.dashboard);s.mux.HandleFunc("GET /ui/",s.dashboard);s.mux.HandleFunc("GET /",s.root);
s.mux.HandleFunc("GET /api/tier",func(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"tier":s.limits.Tier,"upgrade_url":"https://stockyard.dev/billfold/"})})
return s}
func(s *Server)ServeHTTP(w http.ResponseWriter,r *http.Request){s.mux.ServeHTTP(w,r)}
func wj(w http.ResponseWriter,c int,v any){w.Header().Set("Content-Type","application/json");w.WriteHeader(c);json.NewEncoder(w).Encode(v)}
func we(w http.ResponseWriter,c int,m string){wj(w,c,map[string]string{"error":m})}
func(s *Server)root(w http.ResponseWriter,r *http.Request){if r.URL.Path!="/"{http.NotFound(w,r);return};http.Redirect(w,r,"/ui",302)}
func(s *Server)list(w http.ResponseWriter,r *http.Request){
    q:=r.URL.Query().Get("q")
    filters:=map[string]string{}
    if v:=r.URL.Query().Get("status");v!=""{filters["status"]=v}
    if q!=""||len(filters)>0{wj(w,200,map[string]any{"invoices":oe(s.db.Search(q,filters))});return}
    wj(w,200,map[string]any{"invoices":oe(s.db.List())})
}
func(s *Server)create(w http.ResponseWriter,r *http.Request){if s.limits.MaxItems>0{items:=s.db.List();if len(items)>=s.limits.MaxItems{we(w,402,"Free tier limit reached. Upgrade at https://stockyard.dev/billfold/");return}};var e store.Invoice;json.NewDecoder(r.Body).Decode(&e);if e.ClientName==""{we(w,400,"name required");return};s.db.Create(&e);wj(w,201,s.db.Get(e.ID))}
func(s *Server)get(w http.ResponseWriter,r *http.Request){e:=s.db.Get(r.PathValue("id"));if e==nil{we(w,404,"not found");return};wj(w,200,e)}
func(s *Server)update(w http.ResponseWriter,r *http.Request){
    existing:=s.db.Get(r.PathValue("id"));if existing==nil{we(w,404,"not found");return}
    var patch store.Invoice;json.NewDecoder(r.Body).Decode(&patch);patch.ID=existing.ID;patch.CreatedAt=existing.CreatedAt
    if patch.ClientName==""{
patch.ClientName=existing.ClientName};if patch.DueDate==""{
patch.DueDate=existing.DueDate};if patch.Status==""{
patch.Status=existing.Status};if patch.LineItems==""{
patch.LineItems=existing.LineItems};if patch.Notes==""{
patch.Notes=existing.Notes};if patch.PaidAt==""{
patch.PaidAt=existing.PaidAt}
    s.db.Update(&patch);wj(w,200,s.db.Get(patch.ID))
}
func(s *Server)del(w http.ResponseWriter,r *http.Request){s.db.Delete(r.PathValue("id"));wj(w,200,map[string]string{"deleted":"ok"})}
func(s *Server)stats(w http.ResponseWriter,r *http.Request){wj(w,200,s.db.Stats())}
func(s *Server)health(w http.ResponseWriter,r *http.Request){wj(w,200,map[string]any{"status":"ok","service":"billfold","invoices":s.db.Count()})}
func oe[T any](s []T)[]T{if s==nil{return[]T{}};return s}
func init(){log.SetFlags(log.LstdFlags|log.Lshortfile)}
