package utils

import "lehu-video/pkg/password"

// GetPasswordSalt returns a random salt string with the given length
func GetPasswordSalt() (string, error) {
	return password.NewSalt()
}

func GenerateMd5WithSalt(raw, salt string) string {
	return password.MD5WithSalt(raw, salt)
}
