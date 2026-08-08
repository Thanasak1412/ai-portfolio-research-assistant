package password

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"

	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/application"
	"github.com/Thanasak1412/ai-portfolio-research-assistant/backend/internal/identity/domain"
)

const (
	MemoryKiB         = uint32(65536)
	Iterations        = uint32(3)
	Parallelism       = uint8(2)
	SaltLength        = 16
	KeyLength         = uint32(32)
	MinimumCharacters = 12
	MaximumBytes      = 1024
)

type Argon2id struct{}

func New() *Argon2id { return &Argon2id{} }

func (adapter *Argon2id) Hash(ctx context.Context, password string) (domain.PasswordHash, error) {
	if err := validatePassword(password); err != nil {
		return domain.PasswordHash{}, err
	}
	if err := ctx.Err(); err != nil {
		return domain.PasswordHash{}, err
	}
	salt := make([]byte, SaltLength)
	if _, err := rand.Read(salt); err != nil {
		return domain.PasswordHash{}, application.ErrCredentialRejected
	}
	derived := argon2.IDKey([]byte(password), salt, Iterations, MemoryKiB, Parallelism, KeyLength)
	encoded := fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, MemoryKiB, Iterations, Parallelism,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(derived))
	return domain.NewPasswordHash(encoded)
}

func (adapter *Argon2id) Verify(ctx context.Context, password string, stored domain.PasswordHash) (application.PasswordVerification, error) {
	if err := validatePassword(password); err != nil {
		return application.PasswordVerification{}, application.ErrCredentialRejected
	}
	parameters, salt, expected, err := parsePHC(stored.Encoded())
	if err != nil {
		return application.PasswordVerification{}, application.ErrCredentialRejected
	}
	if err := ctx.Err(); err != nil {
		return application.PasswordVerification{}, err
	}
	actual := argon2.IDKey([]byte(password), salt, parameters.iterations, parameters.memory, parameters.parallelism, uint32(len(expected)))
	if subtle.ConstantTimeCompare(actual, expected) != 1 {
		return application.PasswordVerification{}, application.ErrCredentialRejected
	}
	return application.PasswordVerification{Verified: true, NeedsRehash: needsRehash(parameters)}, nil
}

type phcParameters struct {
	memory      uint32
	iterations  uint32
	parallelism uint8
}

func parsePHC(encoded string) (phcParameters, []byte, []byte, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return phcParameters{}, nil, nil, application.ErrCredentialRejected
	}
	var parameters phcParameters
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &parameters.memory, &parameters.iterations, &parameters.parallelism); err != nil {
		return phcParameters{}, nil, nil, application.ErrCredentialRejected
	}
	canonical := "m=" + strconv.FormatUint(uint64(parameters.memory), 10) + ",t=" + strconv.FormatUint(uint64(parameters.iterations), 10) + ",p=" + strconv.FormatUint(uint64(parameters.parallelism), 10)
	if parts[3] != canonical || parameters.memory != MemoryKiB || parameters.iterations != Iterations || parameters.parallelism != Parallelism {
		return phcParameters{}, nil, nil, application.ErrCredentialRejected
	}
	salt, err := base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(salt) != SaltLength {
		return phcParameters{}, nil, nil, application.ErrCredentialRejected
	}
	hash, err := base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(hash) != int(KeyLength) {
		return phcParameters{}, nil, nil, application.ErrCredentialRejected
	}
	return parameters, salt, hash, nil
}

func needsRehash(parameters phcParameters) bool {
	return parameters.memory != MemoryKiB || parameters.iterations != Iterations || parameters.parallelism != Parallelism
}

func validatePassword(password string) error {
	if !utf8.ValidString(password) || utf8.RuneCountInString(password) < MinimumCharacters || len(password) > MaximumBytes {
		return application.ErrCredentialRejected
	}
	return nil
}

var _ application.PasswordService = (*Argon2id)(nil)
