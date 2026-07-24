package credentialstore

import (
	"errors"
	"testing"

	"github.com/zalando/go-keyring"
)

// fakeStore is an in-memory CredentialStore with an optional forced error.
type fakeStore struct {
	m       map[string]string
	failGet bool
	failSet bool
}

func newFakeStore() *fakeStore { return &fakeStore{m: map[string]string{}} }

func (f *fakeStore) GetSecret(key string) (string, error) {
	if f.failGet {
		return "", errors.New("get boom")
	}
	return f.m[key], nil
}

func (f *fakeStore) SetSecret(key, value string) error {
	if f.failSet {
		return errors.New("set boom")
	}
	f.m[key] = value
	return nil
}

func (f *fakeStore) DeleteSecret(key string) error {
	delete(f.m, key)
	return nil
}

func TestChain_GetPrefersPrimary(t *testing.T) {
	primary := newFakeStore()
	fallback := newFakeStore()
	primary.m["k"] = "from-primary"
	fallback.m["k"] = "from-fallback"

	got, err := NewChainStore(primary, fallback).GetSecret("k")
	if err != nil || got != "from-primary" {
		t.Fatalf("got %q err %v, want from-primary", got, err)
	}
}

func TestChain_GetFallsBackWhenPrimaryEmpty(t *testing.T) {
	primary := newFakeStore()
	fallback := newFakeStore()
	fallback.m["k"] = "from-fallback"

	got, _ := NewChainStore(primary, fallback).GetSecret("k")
	if got != "from-fallback" {
		t.Fatalf("got %q, want from-fallback", got)
	}
}

func TestChain_GetFallsBackWhenPrimaryErrors(t *testing.T) {
	primary := &fakeStore{m: map[string]string{}, failGet: true}
	fallback := newFakeStore()
	fallback.m["k"] = "from-fallback"

	got, err := NewChainStore(primary, fallback).GetSecret("k")
	if err != nil || got != "from-fallback" {
		t.Fatalf("got %q err %v, want from-fallback", got, err)
	}
}

func TestChain_SetFallsBackWhenPrimaryErrors(t *testing.T) {
	primary := &fakeStore{m: map[string]string{}, failSet: true}
	fallback := newFakeStore()

	if err := NewChainStore(primary, fallback).SetSecret("k", "v"); err != nil {
		t.Fatalf("SetSecret err %v", err)
	}
	if fallback.m["k"] != "v" {
		t.Fatalf("expected fallback write, got %q", fallback.m["k"])
	}
}

func TestChain_SetSuccessScrubsFallback(t *testing.T) {
	primary := newFakeStore()
	fallback := newFakeStore()
	fallback.m["k"] = "legacy-plaintext"

	if err := NewChainStore(primary, fallback).SetSecret("k", "v"); err != nil {
		t.Fatalf("SetSecret err %v", err)
	}
	if primary.m["k"] != "v" {
		t.Fatalf("expected primary write, got %q", primary.m["k"])
	}
	if _, ok := fallback.m["k"]; ok {
		t.Fatalf("expected fallback scrubbed, still present: %q", fallback.m["k"])
	}
}

func TestChain_DeleteClearsBoth(t *testing.T) {
	primary := newFakeStore()
	fallback := newFakeStore()
	primary.m["k"] = "a"
	fallback.m["k"] = "b"

	if err := NewChainStore(primary, fallback).DeleteSecret("k"); err != nil {
		t.Fatalf("DeleteSecret err %v", err)
	}
	if _, ok := primary.m["k"]; ok {
		t.Fatalf("primary not cleared")
	}
	if _, ok := fallback.m["k"]; ok {
		t.Fatalf("fallback not cleared")
	}
}

func TestKeyring_RoundTrip(t *testing.T) {
	keyring.MockInit()
	s := NewKeyringStore()

	// Absent key returns ("", nil).
	if v, err := s.GetSecret("cx_apikey"); err != nil || v != "" {
		t.Fatalf("absent key got %q err %v", v, err)
	}
	if err := s.SetSecret("cx_apikey", "tok"); err != nil {
		t.Fatalf("SetSecret err %v", err)
	}
	if v, _ := s.GetSecret("cx_apikey"); v != "tok" {
		t.Fatalf("got %q, want tok", v)
	}
	// Delete then idempotent second delete.
	if err := s.DeleteSecret("cx_apikey"); err != nil {
		t.Fatalf("DeleteSecret err %v", err)
	}
	if err := s.DeleteSecret("cx_apikey"); err != nil {
		t.Fatalf("second DeleteSecret should be no-op, got %v", err)
	}
}
