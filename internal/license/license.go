package license

import (
	"crypto/ed25519"
	"fmt"
	"encoding/base64"
	"strings"
)

const publicKey = "3af8f9593b3331c27994f1eeacf111c727ff6015016b0af44ed3ca6934d40b13"

func Validate(key string) bool {
	parts := strings.SplitN(key, ".", 2)
	if len(parts) != 2 {
		return false
	}
	pubBytes, err := hexDecode(publicKey)
	if err != nil || len(pubBytes) != ed25519.PublicKeySize {
		return false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return false
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return false
	}
	return ed25519.Verify(pubBytes, payload, sig)
}

func hexDecode(s string) ([]byte, error) {
	b := make([]byte, len(s)/2)
	for i := range b {
		_, err := hexByte(s[i*2], s[i*2+1], &b[i])
		if err != nil {
			return nil, err
		}
	}
	return b, nil
}

func hexByte(hi, lo byte, out *byte) (int, error) {
	h, err := hexNibble(hi)
	if err != nil {
		return 0, err
	}
	l, err := hexNibble(lo)
	if err != nil {
		return 0, err
	}
	*out = byte(h<<4 | l)
	return 1, nil
}

func hexNibble(c byte) (byte, error) {
	switch {
	case c >= '0' && c <= '9':
		return c - '0', nil
	case c >= 'a' && c <= 'f':
		return c - 'a' + 10, nil
	case c >= 'A' && c <= 'F':
		return c - 'A' + 10, nil
	}
	return 0, fmt.Errorf("invalid hex char %c", c)
}
