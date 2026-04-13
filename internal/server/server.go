package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/stockyard-dev/stockyard-billfold/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

// resourceName is the canonical key for extras storage and the API path.
const resourceName = "invoices"

type Server struct {
	db      *store.DB
	mux     *http.ServeMux
	limMu   sync.RWMutex // guards limits, which can be hot-reloaded by /api/license/activate
	limits  Limits
	dataDir string
	pCfg    map[string]json.RawMessage
	bus     *bus.Bus // optional cross-tool event bus; nil if not configured
}

func New(db *store.DB, limits Limits, dataDir string, b *bus.Bus) *Server {
	s := &Server{
		db:      db,
		mux:     http.NewServeMux(),
		limits:  limits,
		dataDir: dataDir,
		bus:     b,
	}
	s.loadPersonalConfig()

	// Invoice CRUD
	s.mux.HandleFunc("GET /api/invoices", s.list)
	s.mux.HandleFunc("POST /api/invoices", s.create)
	s.mux.HandleFunc("GET /api/invoices/{id}", s.get)
	s.mux.HandleFunc("PUT /api/invoices/{id}", s.update)
	s.mux.HandleFunc("DELETE /api/invoices/{id}", s.del)

	// Stats / health
	s.mux.HandleFunc("GET /api/stats", s.stats)
	s.mux.HandleFunc("GET /api/health", s.health)

	// Personalization
	s.mux.HandleFunc("GET /api/config", s.configHandler)

	// Extras (custom fields)
	s.mux.HandleFunc("GET /api/extras/{resource}", s.listExtras)
	s.mux.HandleFunc("GET /api/extras/{resource}/{id}", s.getExtras)
	s.mux.HandleFunc("PUT /api/extras/{resource}/{id}", s.putExtras)

	// License activation — accepts a key, validates, persists, hot-reloads tier
	s.mux.HandleFunc("POST /api/license/activate", s.activateLicense)

	// Dashboard
	s.mux.HandleFunc("GET /ui", s.dashboard)
	s.mux.HandleFunc("GET /ui/", s.dashboard)
	s.mux.HandleFunc("GET /", s.root)

	// Tier — read-only license info for dashboard banner. Always reachable.
	s.mux.HandleFunc("GET /api/tier", s.tierInfo)

	s.subscribeBus()
	return s
}

// ServeHTTP wraps the underlying mux with a license-gate middleware.
// In trial-required mode, all writes (POST/PUT/DELETE/PATCH) return 402
// EXCEPT POST /api/license/activate (the only way out of trial state).
// Reads are always allowed — the brand promise is that data on disk
// stays accessible even without an active license.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.shouldBlockWrite(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusPaymentRequired)
		w.Write([]byte(`{"error":"Trial required. Start a 14-day free trial at https://stockyard.dev/ — or paste an existing license key in the dashboard under \"Activate License\".","tier":"trial-required"}`))
		return
	}
	s.mux.ServeHTTP(w, r)
}

// shouldBlockWrite returns true when the current tier is trial-required
// AND the incoming request is a non-allowlisted write.
func (s *Server) shouldBlockWrite(r *http.Request) bool {
	s.limMu.RLock()
	tier := s.limits.Tier
	s.limMu.RUnlock()
	if tier != "trial-required" {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	switch r.URL.Path {
	case "/api/license/activate":
		return false
	}
	return true
}

// activateLicense accepts {license_key: "SY-..."}, validates, persists
// to dataDir/license.txt, and hot-reloads s.limits.
func (s *Server) activateLicense(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024))
	if err != nil {
		we(w, 400, "could not read request body")
		return
	}
	var req struct {
		LicenseKey string `json:"license_key"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		we(w, 400, "invalid json: "+err.Error())
		return
	}
	key := strings.TrimSpace(req.LicenseKey)
	if key == "" {
		we(w, 400, "license_key is required")
		return
	}
	if !ValidateLicenseKey(key) {
		we(w, 400, "license key is not valid for this product — make sure you copied the entire key from the welcome email, including the SY- prefix")
		return
	}
	if err := PersistLicense(s.dataDir, key); err != nil {
		log.Printf("billfold: license persist failed: %v", err)
		we(w, 500, "could not save the license key to disk: "+err.Error())
		return
	}
	s.limMu.Lock()
	s.limits = ProLimits()
	s.limMu.Unlock()
	log.Printf("billfold: license activated via dashboard, persisted to %s/%s", s.dataDir, licenseFilename)
	wj(w, 200, map[string]any{
		"ok":   true,
		"tier": "pro",
	})
}

// tierInfo returns the current tier and a hint for the dashboard banner.
func (s *Server) tierInfo(w http.ResponseWriter, r *http.Request) {
	s.limMu.RLock()
	tier := s.limits.Tier
	s.limMu.RUnlock()
	resp := map[string]any{
		"tier": tier,
	}
	if tier == "trial-required" {
		resp["trial_required"] = true
		resp["start_trial_url"] = "https://stockyard.dev/"
		resp["message"] = "Your trial is not active. Reads work, but you cannot add or change invoices until you start a 14-day trial or activate an existing license key."
	} else {
		resp["trial_required"] = false
	}
	wj(w, 200, resp)
}

// ─── helpers ──────────────────────────────────────────────────────

func wj(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func we(w http.ResponseWriter, code int, msg string) {
	wj(w, code, map[string]string{"error": msg})
}

func oe[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	http.Redirect(w, r, "/ui", 302)
}

// ─── personalization ──────────────────────────────────────────────

func (s *Server) loadPersonalConfig() {
	path := filepath.Join(s.dataDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var cfg map[string]json.RawMessage
	if err := json.Unmarshal(data, &cfg); err != nil {
		log.Printf("billfold: warning: could not parse config.json: %v", err)
		return
	}
	s.pCfg = cfg
	log.Printf("billfold: loaded personalization from %s", path)
}

func (s *Server) configHandler(w http.ResponseWriter, r *http.Request) {
	if s.pCfg == nil {
		wj(w, 200, map[string]any{})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.pCfg)
}

// ─── extras ───────────────────────────────────────────────────────

func (s *Server) listExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	all := s.db.AllExtras(resource)
	out := make(map[string]json.RawMessage, len(all))
	for id, data := range all {
		out[id] = json.RawMessage(data)
	}
	wj(w, 200, out)
}

func (s *Server) getExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	data := s.db.GetExtras(resource, id)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data))
}

func (s *Server) putExtras(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	id := r.PathValue("id")
	body, err := io.ReadAll(r.Body)
	if err != nil {
		we(w, 400, "read body")
		return
	}
	var probe map[string]any
	if err := json.Unmarshal(body, &probe); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if err := s.db.SetExtras(resource, id, string(body)); err != nil {
		we(w, 500, "save failed")
		return
	}
	wj(w, 200, map[string]string{"ok": "saved"})
}

// ─── Invoice CRUD ──────────────────────────────────────────────

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	filters := map[string]string{}
	if v := r.URL.Query().Get("status"); v != "" {
		filters["status"] = v
	}
	if q != "" || len(filters) > 0 {
		wj(w, 200, map[string]any{"invoices": oe(s.db.Search(q, filters))})
		return
	}
	wj(w, 200, map[string]any{"invoices": oe(s.db.List())})
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	var e store.Invoice
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		we(w, 400, "invalid json")
		return
	}
	if e.ClientName == "" {
		we(w, 400, "name required")
		return
	}
	if err := s.db.Create(&e); err != nil {
		we(w, 500, "create failed")
		return
	}
	created := s.db.Get(e.ID)
	// Fire invoice.sent only if the invoice is created already in a
	// non-draft state. Most invoices land as 'draft' and transition
	// to 'sent' via an update — that transition is caught below.
	if created != nil && created.Status == "sent" {
		s.publishInvoice("invoice.sent", created)
	}
	wj(w, 201, created)
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	e := s.db.Get(r.PathValue("id"))
	if e == nil {
		we(w, 404, "not found")
		return
	}
	wj(w, 200, e)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	existing := s.db.Get(r.PathValue("id"))
	if existing == nil {
		we(w, 404, "not found")
		return
	}
	var patch store.Invoice
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		we(w, 400, "invalid json")
		return
	}
	patch.ID = existing.ID
	patch.CreatedAt = existing.CreatedAt
	if patch.ClientName == "" {
		patch.ClientName = existing.ClientName
	}
	if patch.DueDate == "" {
		patch.DueDate = existing.DueDate
	}
	if patch.Status == "" {
		patch.Status = existing.Status
	}
	if patch.LineItems == "" {
		patch.LineItems = existing.LineItems
	}
	if patch.Notes == "" {
		patch.Notes = existing.Notes
	}
	if patch.PaidAt == "" {
		patch.PaidAt = existing.PaidAt
	}
	if err := s.db.Update(&patch); err != nil {
		we(w, 500, "update failed")
		return
	}
	updated := s.db.Get(patch.ID)
	// Fire bus events on state transitions only — NOT on every edit.
	// Idempotency: subscribers are expected to be keyed on invoice_id
	// but we still don't want to re-fire e.g. paid→paid over and over.
	if updated != nil && existing.Status != updated.Status {
		switch updated.Status {
		case "sent":
			s.publishInvoice("invoice.sent", updated)
		case "paid":
			s.publishInvoice("invoice.paid", updated)
		case "overdue":
			s.publishInvoice("invoice.overdue", updated)
		}
	}
	wj(w, 200, updated)
}

func (s *Server) del(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.db.Delete(id)
	s.db.DeleteExtras(resourceName, id)
	wj(w, 200, map[string]string{"deleted": "ok"})
}

func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, s.db.Stats())
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	wj(w, 200, map[string]any{
		"status":  "ok",
		"service": "billfold",
		"count":   s.db.Count(),
	})
}

// publishInvoice is the fire-and-forget wrapper around the optional
// bus. No-op when s.bus is nil (standalone mode). Runs in a goroutine
// so HTTP responses never block on bus writes. Errors are logged,
// never surfaced — a failed publish must not break user-facing requests.
//
// Payload shape is locked by docs/BUS-TOPICS.md in stockyard-desktop.
func (s *Server) publishInvoice(topic string, inv *store.Invoice) {
	if s.bus == nil || inv == nil {
		return
	}
	payload := map[string]any{
		"invoice_id":  inv.ID,
		"client_name": inv.ClientName,
		"amount":      inv.Amount,
		"due_date":    inv.DueDate,
		"status":      inv.Status,
		"paid_at":     inv.PaidAt,
	}
	go func() {
		if _, err := s.bus.Publish(topic, payload); err != nil {
			log.Printf("billfold: bus publish %s failed: %v", topic, err)
		}
	}()
}

// subscribeBus wires cross-tool events to auto-drafted invoice actions.
// No-op when s.bus is nil (standalone mode).
//
// Allowlist-only (not SubscribeAll) so unexpected future topics don't
// silently start creating invoices. Expanding this list is a
// PR-reviewed change, not a config flag — every addition is "a new
// way invoices can appear without the user clicking New Invoice."
//
// Idempotency: each handler embeds a marker in invoice Notes or
// LineItems ([quote:<id>], time:<entry_id>) and linear-scans existing
// invoices before creating/mutating. Bus cursor initializes at the
// current high-water mark on Open (see bus.go:199), so process
// restart does NOT replay old events — duplicate fires during a
// single bundle lifetime are the only dedup concern today.
//
// Handlers return nil on decode/data errors: we don't want the bus
// retrying a permanently broken payload, and the bus has no retry
// semantics anyway (see bus.go Handler docstring).
func (s *Server) subscribeBus() {
	if s.bus == nil {
		return
	}
	s.bus.Subscribe("quote.accepted", func(_ context.Context, e bus.Event) error {
		return s.handleQuoteAccepted(e)
	})
	s.bus.Subscribe("time.logged", func(_ context.Context, e bus.Event) error {
		return s.handleTimeLogged(e)
	})
	log.Printf("billfold: subscribed to quote.accepted, time.logged")
}

// handleQuoteAccepted auto-drafts an invoice from an accepted quote.
//
// Shape decisions (see BUS-TOPICS.md):
//   - ClientName = payload.client_name (free text, no contact_id FK
//     in billfold today).
//   - Amount = int(payload.total). Billfold's Amount column is INTEGER;
//     estimate carries total as float64. Decimals are truncated. Unit
//     is whatever the two tools agree on (neither declares currency).
//   - Status = "draft". User reviews before sending.
//   - LineItems = one line referencing the quote: description = quote
//     title, amount = int(total), source = "quote:<quote_id>".
//   - Notes = human-readable provenance including the [quote:<id>]
//     marker used for idempotency.
//   - DueDate = "" (user fills in — we have no policy to infer).
func (s *Server) handleQuoteAccepted(e bus.Event) error {
	var p map[string]any
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		log.Printf("billfold: decode quote.accepted: %v", err)
		return nil
	}
	quoteID := stringField(p, "quote_id")
	if quoteID == "" {
		log.Printf("billfold: quote.accepted missing quote_id, skipping")
		return nil
	}
	marker := fmt.Sprintf("[quote:%s]", quoteID)
	// Idempotency: skip if an invoice already references this quote.
	for _, existing := range s.db.List() {
		if strings.Contains(existing.Notes, marker) {
			log.Printf("billfold: quote.accepted for %s already drafted as invoice %s, skipping", quoteID, existing.ID)
			return nil
		}
	}
	clientName := stringField(p, "client_name")
	total := floatField(p, "total")
	amount := int(total)
	title := stringField(p, "title")
	if title == "" {
		title = "Services from accepted quote"
	}
	lineItems := []map[string]any{{
		"description": title,
		"amount":      amount,
		"source":      "quote:" + quoteID,
	}}
	liJSON, err := json.Marshal(lineItems)
	if err != nil {
		log.Printf("billfold: marshal line items for quote %s: %v", quoteID, err)
		return nil
	}
	notes := fmt.Sprintf("Auto-drafted from accepted quote on %s. %s",
		time.Now().UTC().Format("2006-01-02"), marker)
	inv := store.Invoice{
		ClientName: clientName,
		Amount:     amount,
		Status:     "draft",
		LineItems:  string(liJSON),
		Notes:      notes,
	}
	if err := s.db.Create(&inv); err != nil {
		log.Printf("billfold: create invoice from quote %s: %v", quoteID, err)
		return nil
	}
	log.Printf("billfold: auto-drafted invoice %s from quote %s (client=%q amount=%d)",
		inv.ID, quoteID, clientName, amount)
	return nil
}

// handleTimeLogged appends a line item to a matching draft invoice
// when a billable time entry is logged.
//
// Shape decisions (see BUS-TOPICS.md):
//   - Only billable=true entries trigger the append. Non-billable
//     time is out of scope for invoicing.
//   - Match: the time entry's `project` (free text) is compared
//     case-insensitively + trimmed against existing DRAFT invoices'
//     `client_name`. First match wins (List() returns created_at DESC
//     so this is the most recent draft for that client).
//   - No match = log and drop. We do NOT auto-create an invoice from
//     a time entry alone — that would silently manufacture invoices
//     the user may not intend. The user must already have a draft
//     open for that client for line items to accumulate.
//   - Amount = 0. The payload carries duration in seconds but no
//     rate, and billfold stores no contact-rate map. The user sets
//     the line's amount when they finalize the invoice.
//   - Idempotency: each appended line includes source="time:<id>".
//     Before appending, scan the existing LineItems JSON for this
//     marker and skip if already present.
func (s *Server) handleTimeLogged(e bus.Event) error {
	var p map[string]any
	if err := json.Unmarshal(e.Payload, &p); err != nil {
		log.Printf("billfold: decode time.logged: %v", err)
		return nil
	}
	if !boolField(p, "billable") {
		return nil
	}
	entryID := stringField(p, "entry_id")
	if entryID == "" {
		log.Printf("billfold: time.logged missing entry_id, skipping")
		return nil
	}
	project := strings.TrimSpace(stringField(p, "project"))
	if project == "" {
		log.Printf("billfold: time.logged has empty project, nothing to match against, skipping entry %s", entryID)
		return nil
	}
	needle := strings.ToLower(project)
	var target *store.Invoice
	for _, inv := range s.db.List() {
		if inv.Status != "draft" {
			continue
		}
		if strings.ToLower(strings.TrimSpace(inv.ClientName)) == needle {
			i := inv
			target = &i
			break
		}
	}
	if target == nil {
		log.Printf("billfold: time.logged entry %s (project=%q) has no matching draft invoice, skipping", entryID, project)
		return nil
	}
	marker := "time:" + entryID
	// Parse existing line items. Tolerate empty / "[]" / malformed.
	existingRaw := strings.TrimSpace(target.LineItems)
	var items []map[string]any
	if existingRaw != "" && existingRaw != "[]" {
		if err := json.Unmarshal([]byte(existingRaw), &items); err != nil {
			// Malformed JSON from an older manual edit. Don't clobber
			// the user's data — skip and surface the issue in logs.
			log.Printf("billfold: invoice %s has unparseable line_items, not auto-appending time entry %s: %v", target.ID, entryID, err)
			return nil
		}
	}
	// Idempotency: skip if this entry already contributed a line.
	for _, it := range items {
		if src, _ := it["source"].(string); src == marker {
			return nil
		}
	}
	desc := strings.TrimSpace(stringField(p, "description"))
	if desc == "" {
		desc = strings.TrimSpace(stringField(p, "task"))
	}
	if desc == "" {
		desc = "Time logged"
	}
	durationSec := intField(p, "duration_seconds")
	items = append(items, map[string]any{
		"description":      desc,
		"amount":           0,
		"source":           marker,
		"duration_seconds": durationSec,
	})
	newJSON, err := json.Marshal(items)
	if err != nil {
		log.Printf("billfold: marshal appended line items for invoice %s: %v", target.ID, err)
		return nil
	}
	target.LineItems = string(newJSON)
	if err := s.db.Update(target); err != nil {
		log.Printf("billfold: update invoice %s with time entry %s: %v", target.ID, entryID, err)
		return nil
	}
	log.Printf("billfold: appended time entry %s (%ds) to invoice %s (client=%q)",
		entryID, durationSec, target.ID, target.ClientName)
	return nil
}

// stringField returns m[k] as a string, or "" if absent / wrong type.
func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

// floatField returns m[k] as a float64, tolerating int → float
// coercion (JSON numbers unmarshal as float64 anyway, but cover the
// case of an explicit integer field).
func floatField(m map[string]any, k string) float64 {
	switch v := m[k].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case json.Number:
		f, _ := v.Float64()
		return f
	}
	return 0
}

// intField returns m[k] as an int. JSON numbers are float64 after
// Unmarshal, so we truncate. Booleans, strings, nil → 0.
func intField(m map[string]any, k string) int {
	switch v := m[k].(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	}
	return 0
}

// boolField returns m[k] as a bool. Accepts real bools, 0/1 ints
// (sundial stores billable as int but publishes as bool), and "true"
// string (defensive — nothing should publish this, but cheap to
// tolerate).
func boolField(m map[string]any, k string) bool {
	switch v := m[k].(type) {
	case bool:
		return v
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case string:
		return v == "true" || v == "1"
	}
	return false
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
