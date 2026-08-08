package token

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
)

var keyIDPattern = regexp.MustCompile(`^auth-ed25519-[0-9]{8}-[0-9]{2}$`)

type VerificationKeyInput struct {
	KeyID        string `json:"kid"`
	PublicKeyB64 string `json:"publicKeyB64"`
}

type KeyRingInput struct {
	ActiveKeyID         string                 `json:"activeKid"`
	ActivePrivateKeyB64 string                 `json:"activePrivateKeyB64"`
	VerificationKeys    []VerificationKeyInput `json:"verificationKeys"`
}

type KeyRing struct {
	activeID string
	private  ed25519.PrivateKey
	public   map[string]ed25519.PublicKey
}

func LoadEnvironmentKeyRing(activeKeyID string, privateKeyB64 string, verificationKeysJSON string) (*KeyRing, error) {
	verificationKeys, err := ParseVerificationKeysJSON(verificationKeysJSON)
	if err != nil {
		return nil, err
	}
	return ParseKeyRing(KeyRingInput{
		ActiveKeyID: activeKeyID, ActivePrivateKeyB64: privateKeyB64, VerificationKeys: verificationKeys,
	})
}

func ParseKeyRing(input KeyRingInput) (*KeyRing, error) {
	if !validKeyID(input.ActiveKeyID) {
		return nil, fieldError("AUTH_JWT_ACTIVE_KID")
	}
	privateDER, err := base64.StdEncoding.Strict().DecodeString(input.ActivePrivateKeyB64)
	if err != nil {
		return nil, fieldError("AUTH_JWT_ACTIVE_PRIVATE_KEY_B64")
	}
	parsedPrivate, err := x509.ParsePKCS8PrivateKey(privateDER)
	if err != nil {
		return nil, fieldError("AUTH_JWT_ACTIVE_PRIVATE_KEY_B64")
	}
	privateKey, ok := parsedPrivate.(ed25519.PrivateKey)
	if !ok {
		return nil, fieldError("AUTH_JWT_ACTIVE_PRIVATE_KEY_B64")
	}
	publicKeys := make(map[string]ed25519.PublicKey, len(input.VerificationKeys))
	fingerprints := map[string]struct{}{}
	for _, item := range input.VerificationKeys {
		if !validKeyID(item.KeyID) {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		if _, exists := publicKeys[item.KeyID]; exists {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		der, decodeErr := base64.StdEncoding.Strict().DecodeString(item.PublicKeyB64)
		if decodeErr != nil {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		parsed, parseErr := x509.ParsePKIXPublicKey(der)
		if parseErr != nil {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		publicKey, valid := parsed.(ed25519.PublicKey)
		if !valid {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		fingerprint := base64.RawStdEncoding.EncodeToString(publicKey)
		if _, duplicate := fingerprints[fingerprint]; duplicate {
			return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
		}
		fingerprints[fingerprint] = struct{}{}
		publicKeys[item.KeyID] = publicKey
	}
	activePublic, exists := publicKeys[input.ActiveKeyID]
	if !exists || !activePublic.Equal(privateKey.Public()) {
		return nil, fieldError("AUTH_JWT_ACTIVE_KID")
	}
	return &KeyRing{activeID: input.ActiveKeyID, private: privateKey, public: publicKeys}, nil
}

func ParseVerificationKeysJSON(value string) ([]VerificationKeyInput, error) {
	var keys []VerificationKeyInput
	decoder := json.NewDecoder(strings.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&keys); err != nil || len(keys) == 0 {
		return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fieldError("AUTH_JWT_VERIFICATION_KEYS_JSON")
	}
	return keys, nil
}

func LoadLocalKeyRing(path string) (*KeyRing, error) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0o077 != 0 {
		return nil, fieldError("AUTH_JWT_LOCAL_KEY_RING_PATH")
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fieldError("AUTH_JWT_LOCAL_KEY_RING_PATH")
	}
	var input KeyRingInput
	decoder := json.NewDecoder(strings.NewReader(string(content)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return nil, fieldError("AUTH_JWT_LOCAL_KEY_RING_PATH")
	}
	return ParseKeyRing(input)
}

func (ring *KeyRing) ActiveKeyID() string            { return ring.activeID }
func (ring *KeyRing) signingKey() ed25519.PrivateKey { return ring.private }
func (ring *KeyRing) verificationKey(keyID string) (ed25519.PublicKey, bool) {
	key, ok := ring.public[keyID]
	return key, ok
}

func validKeyID(value string) bool {
	if !keyIDPattern.MatchString(value) {
		return false
	}
	_, err := time.Parse("20060102", value[len("auth-ed25519-"):len("auth-ed25519-")+8])
	return err == nil
}

func fieldError(field string) error {
	return errors.Join(application.ErrInvalidSecurityConfig, errors.New(field+" is invalid"))
}
