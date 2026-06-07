package license

import (
	"crypto/x509"
	"encoding/pem"
)

// x509MarshalPKIXPublicKey wraps stdlib so the test file doesn't
// need to import x509 + pem.
func x509MarshalPKIXPublicKey(pub any) ([]byte, error) {
	return x509.MarshalPKIXPublicKey(pub)
}

func pemEncodePublicKey(der []byte) string {
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: der,
	}))
}
