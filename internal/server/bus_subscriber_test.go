package server

// Tests for the bus subscriber: quote.accepted → draft invoice, and
// time.logged → append line item to matching draft.
//
// These tests exercise the handler methods directly with synthesized
// bus.Event payloads. Full end-to-end wiring (real bus, real publish,
// poll dispatch) is exercised in stockyard-desktop's orchestrator
// tests; here we focus on the shape contract and idempotency.

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stockyard-dev/stockyard-billfold/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

func newSubscriberServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	db, err := store.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	// bus=nil is fine for direct handler invocation — subscribeBus() is
	// not exercised in these tests, only the handler functions.
	return New(db, ProLimits(), dir, nil)
}

func eventWith(topic, source string, payload map[string]any) bus.Event {
	raw, _ := json.Marshal(payload)
	return bus.Event{Topic: topic, Source: source, Payload: raw}
}

func TestHandleQuoteAccepted_CreatesDraftInvoice(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("quote.accepted", "estimate", map[string]any{
		"quote_id":    "q-100",
		"client_name": "Acme Yoga",
		"total":       450.75,
		"title":       "Monthly retainer",
		"status":      "accepted",
	})
	if err := s.handleQuoteAccepted(e); err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(list))
	}
	inv := list[0]
	if inv.ClientName != "Acme Yoga" {
		t.Errorf("client_name = %q, want %q", inv.ClientName, "Acme Yoga")
	}
	if inv.Amount != 450 {
		t.Errorf("amount = %d, want 450 (truncated from 450.75)", inv.Amount)
	}
	if inv.Status != "draft" {
		t.Errorf("status = %q, want draft", inv.Status)
	}
	if !strings.Contains(inv.Notes, "[quote:q-100]") {
		t.Errorf("notes missing idempotency marker: %q", inv.Notes)
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(inv.LineItems), &items); err != nil {
		t.Fatalf("line_items not valid JSON: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}
	if got := items[0]["description"]; got != "Monthly retainer" {
		t.Errorf("line description = %v, want Monthly retainer", got)
	}
	if got := items[0]["source"]; got != "quote:q-100" {
		t.Errorf("line source = %v, want quote:q-100", got)
	}
}

func TestHandleQuoteAccepted_IsIdempotent(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("quote.accepted", "estimate", map[string]any{
		"quote_id":    "q-200",
		"client_name": "Dupe Co",
		"total":       100.0,
		"title":       "Thing",
	})
	_ = s.handleQuoteAccepted(e)
	_ = s.handleQuoteAccepted(e) // replay
	_ = s.handleQuoteAccepted(e) // replay again
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 invoice after 3 fires, got %d", len(list))
	}
}

func TestHandleQuoteAccepted_MissingQuoteIDSkips(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("quote.accepted", "estimate", map[string]any{
		"client_name": "No ID Co",
		"total":       50.0,
	})
	_ = s.handleQuoteAccepted(e)
	if n := len(s.db.List()); n != 0 {
		t.Fatalf("expected 0 invoices, got %d (should skip without quote_id)", n)
	}
}

func TestHandleQuoteAccepted_EmptyTitleFallsBack(t *testing.T) {
	s := newSubscriberServer(t)
	e := eventWith("quote.accepted", "estimate", map[string]any{
		"quote_id":    "q-notitle",
		"client_name": "Empty Title Co",
		"total":       25.0,
	})
	_ = s.handleQuoteAccepted(e)
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 invoice, got %d", len(list))
	}
	var items []map[string]any
	_ = json.Unmarshal([]byte(list[0].LineItems), &items)
	if got, _ := items[0]["description"].(string); got == "" {
		t.Errorf("expected non-empty fallback description, got empty string")
	}
}

func TestHandleTimeLogged_AppendsToMatchingDraft(t *testing.T) {
	s := newSubscriberServer(t)
	// Seed a draft invoice for "Acme Yoga".
	seed := store.Invoice{ClientName: "Acme Yoga", Status: "draft", LineItems: "[]"}
	if err := s.db.Create(&seed); err != nil {
		t.Fatalf("seed create: %v", err)
	}
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id":         "t-1",
		"description":      "Wrote website copy",
		"project":          "Acme Yoga",
		"duration_seconds": float64(3600),
		"billable":         true,
	})
	if err := s.handleTimeLogged(e); err != nil {
		t.Fatalf("handler: %v", err)
	}
	list := s.db.List()
	if len(list) != 1 {
		t.Fatalf("expected still 1 invoice (appended, not created), got %d", len(list))
	}
	var items []map[string]any
	if err := json.Unmarshal([]byte(list[0].LineItems), &items); err != nil {
		t.Fatalf("line_items: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 line item, got %d", len(items))
	}
	if items[0]["source"] != "time:t-1" {
		t.Errorf("source = %v, want time:t-1", items[0]["source"])
	}
	if items[0]["description"] != "Wrote website copy" {
		t.Errorf("description = %v, want Wrote website copy", items[0]["description"])
	}
	// duration_seconds will come back as float64 from JSON round-trip
	if d, _ := items[0]["duration_seconds"].(float64); d != 3600 {
		t.Errorf("duration_seconds = %v, want 3600", items[0]["duration_seconds"])
	}
}

func TestHandleTimeLogged_MatchesCaseInsensitive(t *testing.T) {
	s := newSubscriberServer(t)
	seed := store.Invoice{ClientName: "  Acme YOGA  ", Status: "draft", LineItems: "[]"}
	if err := s.db.Create(&seed); err != nil {
		t.Fatalf("seed: %v", err)
	}
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id": "t-2",
		"project":  "acme yoga",
		"billable": true,
	})
	_ = s.handleTimeLogged(e)
	var items []map[string]any
	_ = json.Unmarshal([]byte(s.db.List()[0].LineItems), &items)
	if len(items) != 1 {
		t.Errorf("expected appended line on case-insensitive+trimmed match, got %d items", len(items))
	}
}

func TestHandleTimeLogged_NonBillableSkipped(t *testing.T) {
	s := newSubscriberServer(t)
	seed := store.Invoice{ClientName: "Acme", Status: "draft", LineItems: "[]"}
	_ = s.db.Create(&seed)
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id": "t-nb",
		"project":  "Acme",
		"billable": false,
	})
	_ = s.handleTimeLogged(e)
	var items []map[string]any
	_ = json.Unmarshal([]byte(s.db.List()[0].LineItems), &items)
	if len(items) != 0 {
		t.Errorf("non-billable entry should not append, got %d items", len(items))
	}
}

func TestHandleTimeLogged_NoMatchingDraftSkipped(t *testing.T) {
	s := newSubscriberServer(t)
	// Draft for a different client
	_ = s.db.Create(&store.Invoice{ClientName: "Other Co", Status: "draft", LineItems: "[]"})
	// Paid invoice for the right client — should NOT be matched (status filter).
	_ = s.db.Create(&store.Invoice{ClientName: "Acme", Status: "paid", LineItems: "[]"})
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id": "t-nomatch",
		"project":  "Acme",
		"billable": true,
	})
	_ = s.handleTimeLogged(e)
	// Neither invoice should have gained a line item.
	for _, inv := range s.db.List() {
		var items []map[string]any
		_ = json.Unmarshal([]byte(inv.LineItems), &items)
		if len(items) != 0 {
			t.Errorf("invoice %s (%s/%s) should have no items, got %d",
				inv.ID, inv.ClientName, inv.Status, len(items))
		}
	}
}

func TestHandleTimeLogged_Idempotent(t *testing.T) {
	s := newSubscriberServer(t)
	_ = s.db.Create(&store.Invoice{ClientName: "Acme", Status: "draft", LineItems: "[]"})
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id":         "t-idem",
		"project":          "Acme",
		"duration_seconds": float64(1800),
		"billable":         true,
	})
	_ = s.handleTimeLogged(e)
	_ = s.handleTimeLogged(e) // replay
	_ = s.handleTimeLogged(e) // replay
	var items []map[string]any
	_ = json.Unmarshal([]byte(s.db.List()[0].LineItems), &items)
	if len(items) != 1 {
		t.Errorf("expected 1 line item after 3 fires, got %d", len(items))
	}
}

func TestHandleTimeLogged_MalformedLineItemsSkips(t *testing.T) {
	s := newSubscriberServer(t)
	_ = s.db.Create(&store.Invoice{
		ClientName: "Acme",
		Status:     "draft",
		LineItems:  "this is not JSON",
	})
	e := eventWith("time.logged", "sundial", map[string]any{
		"entry_id": "t-mal",
		"project":  "Acme",
		"billable": true,
	})
	if err := s.handleTimeLogged(e); err != nil {
		t.Fatalf("handler should not return error on malformed JSON: %v", err)
	}
	inv := s.db.List()[0]
	if inv.LineItems != "this is not JSON" {
		t.Errorf("user's malformed line_items was clobbered: now %q", inv.LineItems)
	}
}
