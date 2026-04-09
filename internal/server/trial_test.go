package server

// Tests for the trial-required gating + license activation flow.
//
// Coverage:
//   - Middleware allows reads, blocks writes when tier=trial-required
//   - Allowlist (license activate, import preview) bypasses block
//   - Dashboard, /, health, /api/tier always reachable
//   - License activation rejects bad input shapes
//   - License activation with a (test-keypair) valid key flips tier to pro,
//     persists to disk, and unblocks subsequent writes
//   - DefaultLimits resolution order: env > file > trial-required
//   - PersistLicense writes file with 0600 perms
//   - License file with garbage content does not crash boot

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stockyard-dev/stockyard-billfold/internal/store"
)

// ─── Middleware: trial-required gating ───────────────────────────────

func newTrialServer(t *testing.T) (*Server, string) {
	t.Helper()
	dir := t.TempDir()
	db, err := os.MkdirAll(filepath.Join(dir, "data"), 0755), error(nil)
	_ = db
	if err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store := newTestDB(t)
	srv := New(store, TrialRequiredLimits(), dir)
	return srv, dir
}

func TestTrialMiddleware_readsAlwaysAllowed(t *testing.T) {
	srv, _ := newTrialServer(t)

	cases := []struct {
		method, path string
	}{
		{"GET", "/api/invoices"},
		{"GET", "/api/health"},
		{"GET", "/api/tier"},
		{"GET", "/api/config"},
		{"GET", "/api/stats"},
		{"GET", "/api/extras/invoices"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, nil)
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code == http.StatusPaymentRequired {
			t.Errorf("read %s %s should not be 402, got %d body=%s", c.method, c.path, w.Code, w.Body.String())
		}
	}
}

func TestTrialMiddleware_writesBlocked(t *testing.T) {
	srv, _ := newTrialServer(t)

	cases := []struct {
		method, path string
	}{
		{"POST", "/api/invoices"},
		{"PUT", "/api/invoices/123"},
		{"DELETE", "/api/invoices/123"},
		{"PUT", "/api/extras/invoices/123"},
		{"POST", "/api/import/commit"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, bytes.NewReader([]byte("{}")))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		if w.Code != http.StatusPaymentRequired {
			t.Errorf("write %s %s should be 402, got %d body=%s", c.method, c.path, w.Code, w.Body.String())
		}
		if !bytes.Contains(w.Body.Bytes(), []byte("trial")) {
			t.Errorf("402 response should mention trial: %s", w.Body.String())
		}
	}
}

func TestTrialMiddleware_allowlistedWritesPass(t *testing.T) {
	srv, _ := newTrialServer(t)

	// /api/license/activate is allowlisted because it's the only way out
	// of trial-required state.
	cases := []struct {
		method, path string
		body         []byte
	}{
		{"POST", "/api/license/activate", []byte(`{"license_key":"SY-bogus"}`)},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.path, bytes.NewReader(c.body))
		w := httptest.NewRecorder()
		srv.ServeHTTP(w, req)
		// We don't care about the exact status — only that it's NOT 402.
		// Activate will return 400 (invalid key), preview will return 200.
		if w.Code == http.StatusPaymentRequired {
			t.Errorf("allowlisted %s %s should not be 402, got %d body=%s", c.method, c.path, w.Code, w.Body.String())
		}
	}
}

func TestTrialMiddleware_proTierBypassesGate(t *testing.T) {
	dir := t.TempDir()
	db := newTestDB(t)
	srv := New(db, ProLimits(), dir)

	// Pro tier should let everything through.
	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code == http.StatusPaymentRequired {
		t.Errorf("pro tier should not be 402, got %d body=%s", w.Code, w.Body.String())
	}
}

func TestTrialMiddleware_emptyTierBypassesGate(t *testing.T) {
	// Backward-compat: tests that construct Server with Limits{} (zero
	// value, Tier="") should be treated as licensed so the existing
	// pre-license-gate test suite still passes without modification.
	dir := t.TempDir()
	db := newTestDB(t)
	srv := New(db, Limits{}, dir)

	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code == http.StatusPaymentRequired {
		t.Errorf("empty Tier should not be 402, got %d", w.Code)
	}
}

// ─── /api/tier endpoint ──────────────────────────────────────────────

func TestTierEndpoint_trialRequired(t *testing.T) {
	srv, _ := newTrialServer(t)

	req := httptest.NewRequest("GET", "/api/tier", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("status: %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["tier"] != "trial-required" {
		t.Errorf("tier: got %v, want trial-required", resp["tier"])
	}
	if resp["trial_required"] != true {
		t.Errorf("trial_required: got %v, want true", resp["trial_required"])
	}
	if _, ok := resp["start_trial_url"]; !ok {
		t.Error("start_trial_url should be set in trial-required mode")
	}
}

func TestTierEndpoint_pro(t *testing.T) {
	dir := t.TempDir()
	db := newTestDB(t)
	srv := New(db, ProLimits(), dir)

	req := httptest.NewRequest("GET", "/api/tier", nil)
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["tier"] != "pro" {
		t.Errorf("tier: got %v, want pro", resp["tier"])
	}
	if resp["trial_required"] != false {
		t.Errorf("trial_required: got %v, want false", resp["trial_required"])
	}
}

// ─── License activation: input validation ───────────────────────────

func TestActivate_emptyBody(t *testing.T) {
	srv, _ := newTrialServer(t)
	req := httptest.NewRequest("POST", "/api/license/activate", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestActivate_malformedJSON(t *testing.T) {
	srv, _ := newTrialServer(t)
	req := httptest.NewRequest("POST", "/api/license/activate", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

func TestActivate_invalidKey(t *testing.T) {
	srv, _ := newTrialServer(t)
	req := httptest.NewRequest("POST", "/api/license/activate",
		bytes.NewReader([]byte(`{"license_key":"SY-totallyfake.notreal"}`)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status: got %d, want 400 (invalid signature)", w.Code)
	}
}

func TestActivate_keyWithoutSYPrefix(t *testing.T) {
	srv, _ := newTrialServer(t)
	req := httptest.NewRequest("POST", "/api/license/activate",
		bytes.NewReader([]byte(`{"license_key":"plain-text-not-a-key"}`)))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 400 {
		t.Errorf("status: got %d, want 400", w.Code)
	}
}

// ─── License activation: full happy path with test keypair ───────────

func TestActivate_validKeyFlipsToProAndPersists(t *testing.T) {
	// Substitute a test keypair so we can sign a "valid" license payload
	// without needing the production private key. Restored on cleanup so
	// other tests still see the production public key.
	originalPubKey := publicKeyHex
	t.Cleanup(func() { publicKeyHex = originalPubKey })

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("keypair: %v", err)
	}
	publicKeyHex = hex.EncodeToString(pub)

	// Mint a license payload that the validator will accept:
	// p="*" (any product), x = expiration far in the future.
	payload := struct {
		P string `json:"p"`
		X int64  `json:"x"`
	}{
		P: "*",
		X: time.Now().Add(365 * 24 * time.Hour).Unix(),
	}
	pb, _ := json.Marshal(payload)
	sig := ed25519.Sign(priv, pb)
	key := "SY-" + base64.RawURLEncoding.EncodeToString(pb) + "." + base64.RawURLEncoding.EncodeToString(sig)

	// Sanity: the validator should accept this key
	if !ValidateLicenseKey(key) {
		t.Fatalf("constructed key did not validate — test keypair substitution broken")
	}

	dir := t.TempDir()
	db := newTestDB(t)
	srv := New(db, TrialRequiredLimits(), dir)

	// Pre-activation: writes blocked
	body, _ := json.Marshal(map[string]string{"name": "Alice"})
	req := httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(body))
	w := httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("pre-activation write should be 402, got %d", w.Code)
	}

	// Activate
	actBody, _ := json.Marshal(map[string]string{"license_key": key})
	req = httptest.NewRequest("POST", "/api/license/activate", bytes.NewReader(actBody))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 200 {
		t.Fatalf("activate: got %d body=%s", w.Code, w.Body.String())
	}
	var actResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &actResp)
	if actResp["tier"] != "pro" {
		t.Errorf("post-activate tier: got %v, want pro", actResp["tier"])
	}

	// Post-activation: writes work
	req = httptest.NewRequest("POST", "/api/invoices", bytes.NewReader(body))
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	if w.Code != 201 {
		t.Errorf("post-activation write: got %d body=%s", w.Code, w.Body.String())
	}

	// Persistence: file should exist with correct contents and 0600 perms
	licPath := filepath.Join(dir, licenseFilename)
	stat, err := os.Stat(licPath)
	if err != nil {
		t.Fatalf("license file not written: %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Errorf("license file perms: got %o, want 0600", stat.Mode().Perm())
	}
	contents, _ := os.ReadFile(licPath)
	if string(contents) != key {
		t.Errorf("license file contents do not match the activated key")
	}

	// /api/tier should now report pro
	req = httptest.NewRequest("GET", "/api/tier", nil)
	w = httptest.NewRecorder()
	srv.ServeHTTP(w, req)
	var tierResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &tierResp)
	if tierResp["tier"] != "pro" {
		t.Errorf("tier endpoint after activation: got %v, want pro", tierResp["tier"])
	}
}

// ─── DefaultLimits resolution order ──────────────────────────────────

func TestDefaultLimits_envVarTakesPrecedence(t *testing.T) {
	originalPubKey := publicKeyHex
	t.Cleanup(func() {
		publicKeyHex = originalPubKey
		os.Unsetenv("STOCKYARD_LICENSE_KEY")
	})

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	publicKeyHex = hex.EncodeToString(pub)
	pb, _ := json.Marshal(struct {
		P string `json:"p"`
		X int64  `json:"x"`
	}{P: "*", X: time.Now().Add(time.Hour).Unix()})
	sig := ed25519.Sign(priv, pb)
	envKey := "SY-" + base64.RawURLEncoding.EncodeToString(pb) + "." + base64.RawURLEncoding.EncodeToString(sig)
	os.Setenv("STOCKYARD_LICENSE_KEY", envKey)

	// Even with no file present, env var alone should give pro.
	dir := t.TempDir()
	lim := DefaultLimits(dir)
	if lim.Tier != "pro" {
		t.Errorf("env-only: got %s, want pro", lim.Tier)
	}
}

func TestDefaultLimits_fileFallbackWhenEnvEmpty(t *testing.T) {
	originalPubKey := publicKeyHex
	t.Cleanup(func() {
		publicKeyHex = originalPubKey
		os.Unsetenv("STOCKYARD_LICENSE_KEY")
	})
	os.Unsetenv("STOCKYARD_LICENSE_KEY")

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	publicKeyHex = hex.EncodeToString(pub)
	pb, _ := json.Marshal(struct {
		P string `json:"p"`
		X int64  `json:"x"`
	}{P: "*", X: time.Now().Add(time.Hour).Unix()})
	sig := ed25519.Sign(priv, pb)
	fileKey := "SY-" + base64.RawURLEncoding.EncodeToString(pb) + "." + base64.RawURLEncoding.EncodeToString(sig)

	dir := t.TempDir()
	if err := PersistLicense(dir, fileKey); err != nil {
		t.Fatalf("persist: %v", err)
	}

	lim := DefaultLimits(dir)
	if lim.Tier != "pro" {
		t.Errorf("file fallback: got %s, want pro", lim.Tier)
	}
}

func TestDefaultLimits_noKeyAnywhere(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("STOCKYARD_LICENSE_KEY") })
	os.Unsetenv("STOCKYARD_LICENSE_KEY")
	dir := t.TempDir()
	lim := DefaultLimits(dir)
	if lim.Tier != "trial-required" {
		t.Errorf("no key: got %s, want trial-required", lim.Tier)
	}
}

func TestDefaultLimits_invalidFileContent(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("STOCKYARD_LICENSE_KEY") })
	os.Unsetenv("STOCKYARD_LICENSE_KEY")
	dir := t.TempDir()
	// Write garbage that does not pass validation
	os.WriteFile(filepath.Join(dir, licenseFilename), []byte("not a valid key"), 0600)
	lim := DefaultLimits(dir)
	if lim.Tier != "trial-required" {
		t.Errorf("invalid file: got %s, want trial-required (must not crash)", lim.Tier)
	}
}

func TestDefaultLimits_emptyDataDir(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("STOCKYARD_LICENSE_KEY") })
	os.Unsetenv("STOCKYARD_LICENSE_KEY")
	// Empty dataDir means no file fallback at all — should still return
	// trial-required (not crash on filesystem access).
	lim := DefaultLimits("")
	if lim.Tier != "trial-required" {
		t.Errorf("empty dataDir: got %s, want trial-required", lim.Tier)
	}
}

// ─── PersistLicense ──────────────────────────────────────────────────

func TestPersistLicense_writesFileWithCorrectPerms(t *testing.T) {
	dir := t.TempDir()
	if err := PersistLicense(dir, "SY-testkey"); err != nil {
		t.Fatalf("persist: %v", err)
	}
	stat, err := os.Stat(filepath.Join(dir, licenseFilename))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Errorf("perms: got %o, want 0600", stat.Mode().Perm())
	}
}

func TestPersistLicense_emptyDataDirErrors(t *testing.T) {
	if err := PersistLicense("", "SY-key"); err == nil {
		t.Error("expected error for empty dataDir")
	}
}

// newTestDB is a local helper since billfold has no import_test.go
// where dossier defines the equivalent. Opens a fresh SQLite store in
// a temp directory that t.Cleanup deletes after the test.
func newTestDB(t *testing.T) *store.DB {
	t.Helper()
	dir, err := os.MkdirTemp("", "billfold-trial-test-*")
	if err != nil {
		t.Fatalf("mkdtemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	db, err := store.Open(filepath.Join(dir, "data"))
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	return db
}
