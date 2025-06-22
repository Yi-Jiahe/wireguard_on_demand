package wireguard

import (
	"crypto/rand"
	"encoding/base64"

	"golang.org/x/crypto/curve25519"
)

// GenerateWireGuardKeyPair returns (privateKeyBase64, publicKeyBase64)
func GenerateWireGuardKeyPair() (string, string, error) {
	var privateKeyBytes [32]byte
	_, err := rand.Read(privateKeyBytes[:])
	if err != nil {
		return "", "", err
	}

	// Clamp the private key as required by X25519
	privateKeyBytes[0] &= 248
	privateKeyBytes[31] &= 127
	privateKeyBytes[31] |= 64

	publicKeyBytes, err := curve25519.X25519(privateKeyBytes[:], curve25519.Basepoint)
	if err != nil {
		return "", "", err
	}

	priv := base64.StdEncoding.EncodeToString(privateKeyBytes[:])
	pub := base64.StdEncoding.EncodeToString(publicKeyBytes)

	return priv, pub, nil
}
