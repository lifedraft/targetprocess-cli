package config

import (
	"errors"

	"github.com/zalando/go-keyring"
)

const (
	keyringService = "targetprocess-cli"
	keyringUser    = "token"
)

// ErrKeyringUnavailable indicates the OS keyring is not accessible.
var ErrKeyringUnavailable = errors.New("keyring unavailable")

// keyringGet retrieves the token from the OS keyring.
// Returns ErrKeyringUnavailable if the keyring cannot be accessed.
func keyringGet() (string, error) {
	token, err := keyring.Get(keyringService, keyringUser)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", nil
		}
		return "", ErrKeyringUnavailable
	}
	return token, nil
}

// keyringSet stores the token in the OS keyring.
// Returns ErrKeyringUnavailable if the keyring cannot be accessed.
func keyringSet(token string) error {
	err := keyring.Set(keyringService, keyringUser, token)
	if err != nil {
		return ErrKeyringUnavailable
	}
	return nil
}

// keyringDelete removes the token from the OS keyring.
func keyringDelete() error {
	err := keyring.Delete(keyringService, keyringUser)
	if err != nil && !errors.Is(err, keyring.ErrNotFound) {
		return ErrKeyringUnavailable
	}
	return nil
}
