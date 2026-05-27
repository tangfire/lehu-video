package password

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	algorithmArgon2id = "argon2id"
	argonTime         = uint32(1)
	argonMemory       = uint32(64 * 1024)
	argonThreads      = uint8(4)
	argonKeyLen       = uint32(32)
	argonSaltLen      = 16
)

func NewSalt() (string, error) {
	b := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func MD5WithSalt(raw, salt string) string {
	hash := md5.Sum([]byte(raw + salt))
	return hex.EncodeToString(hash[:])
}

func Hash(raw string) (encoded string, salt string, err error) {
	saltBytes := make([]byte, argonSaltLen)
	if _, err := io.ReadFull(rand.Reader, saltBytes); err != nil {
		return "", "", err
	}
	key := argon2.IDKey([]byte(raw), saltBytes, argonTime, argonMemory, argonThreads, argonKeyLen)
	encoded = fmt.Sprintf("$%s$v=%d$m=%d,t=%d,p=%d$%s$%s",
		algorithmArgon2id,
		argon2.Version,
		argonMemory,
		argonTime,
		argonThreads,
		base64.RawStdEncoding.EncodeToString(saltBytes),
		base64.RawStdEncoding.EncodeToString(key),
	)
	return encoded, "", nil
}

func Verify(raw, encoded, legacySalt string) (ok bool, needsUpgrade bool) {
	if strings.HasPrefix(encoded, "$"+algorithmArgon2id+"$") {
		ok, err := verifyArgon2id(raw, encoded)
		return ok && err == nil, false
	}
	return MD5WithSalt(raw, legacySalt) == encoded, true
}

func IsModern(encoded string) bool {
	return strings.HasPrefix(encoded, "$"+algorithmArgon2id+"$")
}

func verifyArgon2id(raw, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != algorithmArgon2id {
		return false, errors.New("invalid argon2id hash")
	}
	versionPart := strings.TrimPrefix(parts[2], "v=")
	version, err := strconv.Atoi(versionPart)
	if err != nil || version != argon2.Version {
		return false, errors.New("unsupported argon2id version")
	}
	var memory, timeParam uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeParam, &threads); err != nil {
		return false, err
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, err
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, err
	}
	actual := argon2.IDKey([]byte(raw), salt, timeParam, memory, threads, uint32(len(expected)))
	return string(actual) == string(expected), nil
}
