package server

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// publicKeyHex is the Ed25519 public key used to verify license signatures.
// It is a var (not const) only so tests can substitute a test keypair —
// production code never reassigns it. The current value is the same key
// used by the Stockyard checkout webhook to sign license payloads.
var publicKeyHex = "3af8f9593b3331c27994f1eeacf111c727ff6015016b0af44ed3ca6934d40b13"

// licenseFilename is the per-data-dir fallback location for the license
// key when STOCKYARD_LICENSE_KEY env var is not set. This lets a customer
// activate their license once via POST /api/license/activate and have it
// persist across reboots without rewriting their shell rc.
const licenseFilename = "license.txt"

// Limits describes what the running tool is allowed to do. There is NO
// free tier — Stockyard sells a $7.99/mo bundle with a 14-day paid trial,
// and the open-core proxy is the only thing that runs without a license.
//
// Tier semantics:
//   - "pro"             — license valid, unlimited reads and writes
//   - "trial-required"  — no license; reads allowed, writes return 402,
//     dashboard shows a banner directing the user to
//     start a trial OR paste a key they already have
//   - ""                — used by tests to bypass the license middleware;
//     treated as licensed for backward-compat with
//     pre-license-gate tests
type Limits struct {
	MaxItems int    // 0 = no cap; reserved for future per-tier limits
	Tier     string // "pro" | "trial-required" | ""
}

// TrialRequiredLimits is the default state when no valid license is
// present. There is no item cap — capping items would punish customers
// who pasted the wrong key or whose env var got nuked, and the brand
// promise is that data on disk stays accessible. The write block is
// enforced by the license middleware in server.go, not by an item count.
func TrialRequiredLimits() Limits {
	return Limits{MaxItems: 0, Tier: "trial-required"}
}

// ProLimits is the licensed state. No item cap, no write block.
func ProLimits() Limits {
	return Limits{MaxItems: 0, Tier: "pro"}
}

// DefaultLimits resolves the license key from (1) the STOCKYARD_LICENSE_KEY
// env var, then (2) the licenseFilename in the data directory. The data-dir
// fallback exists so a customer can activate their license once via the
// dashboard's "already have a key?" inline input and have it persist across
// reboots without remembering to set a shell variable.
//
// dataDir may be empty (callers without a data dir, e.g. tests) — in that
// case the file fallback is skipped and only the env var matters.
func DefaultLimits(dataDir string) Limits {
	key := strings.TrimSpace(os.Getenv("STOCKYARD_LICENSE_KEY"))
	source := "env"
	if key == "" && dataDir != "" {
		if data, err := os.ReadFile(filepath.Join(dataDir, licenseFilename)); err == nil {
			key = strings.TrimSpace(string(data))
			source = "file"
		}
	}
	if key == "" {
		log.Printf("[license] no license key — trial required (writes blocked, reads allowed)")
		log.Printf("[license] start a trial: https://stockyard.dev/")
		log.Printf("[license] already have a key? open the dashboard and paste it under \"Activate License\"")
		return TrialRequiredLimits()
	}
	if validateLicenseKey(key, "billfold") {
		log.Printf("[license] valid license loaded from %s — unlocked", source)
		return ProLimits()
	}
	log.Printf("[license] license key from %s did not validate — trial required", source)
	return TrialRequiredLimits()
}

// PersistLicense writes a license key to the data directory so it survives
// reboots without the env var. Called by the activation handler after the
// key has been validated. Returns an error only if the write fails.
func PersistLicense(dataDir, key string) error {
	if dataDir == "" {
		return os.ErrInvalid
	}
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return err
	}
	// 0600 because the license key is a credential — anyone reading it
	// off disk can use it as if they were the customer.
	return os.WriteFile(filepath.Join(dataDir, licenseFilename), []byte(key), 0600)
}

// ValidateLicenseKey is the exported wrapper for the license validation
// logic, used by the activation handler before persisting a key.
func ValidateLicenseKey(key string) bool {
	return validateLicenseKey(key, "billfold")
}

func LimitReached(limit, current int) bool {
	if limit == 0 {
		return false
	}
	return current >= limit
}

func validateLicenseKey(key, product string) bool {
	if !strings.HasPrefix(key, "SY-") {
		return false
	}
	key = key[3:]
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false
	}
	pb, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sb, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(sb) != ed25519.SignatureSize {
		return false
	}
	pk, _ := hexDec(publicKeyHex)
	if len(pk) != ed25519.PublicKeySize {
		return false
	}
	if !ed25519.Verify(ed25519.PublicKey(pk), pb, sb) {
		return false
	}
	var p struct {
		P string `json:"p"`
		X int64  `json:"x"`
	}
	if err := json.Unmarshal(pb, &p); err != nil {
		return false
	}
	if p.X > 0 && time.Now().Unix() > p.X {
		return false
	}
	if p.P != "*" && p.P != "stockyard" && p.P != product {
		return false
	}
	return true
}

func hexDec(s string) ([]byte, error) {
	if len(s)%2 != 0 {
		return nil, os.ErrInvalid
	}
	b := make([]byte, len(s)/2)
	for i := 0; i < len(s); i += 2 {
		h, l := hv(s[i]), hv(s[i+1])
		if h == 255 || l == 255 {
			return nil, os.ErrInvalid
		}
		b[i/2] = h<<4 | l
	}
	return b, nil
}
func hv(c byte) byte {
	switch {
	case c >= '0' && c <= '9':
		return c - '0'
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10
	}
	return 255
}
