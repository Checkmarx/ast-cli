package credentialstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	keyring "github.com/zalando/go-keyring"
)

// keyringOpTimeout bounds each blocking keyring call. Generous enough for
// interactive keychain prompts (macOS re-prompts after every binary upgrade
// when access ACLs reset) while still failing closed eventually.
const keyringOpTimeout = 30 * time.Second

var (
	deniedIndicators      = []string{"denied", "authentication"}
	unavailableIndicators = []string{"not available", "unavailable", "dbus", "failed to connect", "timeout"}
)

type osKeyring struct{}

func (osKeyring) Get(ctx context.Context, service, account string) (string, error) {
	type result struct {
		value string
		err   error
	}
	ch := make(chan result, 1)
	ctx, cancel := context.WithTimeout(ctx, keyringOpTimeout)
	defer cancel()
	go func() {
		value, err := keyring.Get(service, account)
		ch <- result{value: value, err: err}
	}()
	select {
	case res := <-ch:
		if errors.Is(res.err, keyring.ErrNotFound) || (res.err == nil && res.value == "") {
			return "", ErrNotFound
		}
		if res.err != nil {
			return "", mapKeyringError(res.err)
		}
		return res.value, nil
	case <-ctx.Done():
		return "", fmt.Errorf("%w: %w", ErrKeyringUnavailable, ctx.Err())
	}
}

func (osKeyring) Set(ctx context.Context, service, account, value string) error {
	return runKeyringWrite(ctx, func() error {
		return keyring.Set(service, account, value)
	})
}

func (osKeyring) Delete(ctx context.Context, service, account string) error {
	return runKeyringWrite(ctx, func() error {
		return keyring.Delete(service, account)
	})
}

func runKeyringWrite(ctx context.Context, op func() error) error {
	errCh := make(chan error, 1)
	ctx, cancel := context.WithTimeout(ctx, keyringOpTimeout)
	defer cancel()
	go func() {
		errCh <- op()
	}()
	select {
	case err := <-errCh:
		return mapKeyringError(err)
	case <-ctx.Done():
		return fmt.Errorf("%w: %w", ErrKeyringUnavailable, ctx.Err())
	}
}

func mapKeyringError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, keyring.ErrNotFound) {
		return ErrNotFound
	}
	message := strings.ToLower(err.Error())
	switch {
	case containsAny(message, deniedIndicators):
		return fmt.Errorf("%w: %w", ErrAccessDenied, err)
	case containsAny(message, unavailableIndicators):
		return fmt.Errorf("%w: %w", ErrKeyringUnavailable, err)
	default:
		return fmt.Errorf("%w: %w", ErrKeyringFailure, err)
	}
}

func containsAny(message string, indicators []string) bool {
	for _, indicator := range indicators {
		if strings.Contains(message, indicator) {
			return true
		}
	}
	return false
}
