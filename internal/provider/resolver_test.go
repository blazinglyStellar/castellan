package provider

import (
	"context"
	"errors"
	"testing"
)

type mockResolver struct {
	baseURL string
	err     error
}

func (m *mockResolver) ResolveBaseURL(_ context.Context, _ string) (string, error) {
	return m.baseURL, m.err
}

func TestMockResolver_ReturnsURL(t *testing.T) {
	expected := "https://api.example.com/v1"
	m := &mockResolver{baseURL: expected}

	got, err := m.ResolveBaseURL(context.Background(), "any-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

func TestMockResolver_ReturnsError(t *testing.T) {
	expected := errors.New("provider not found")
	m := &mockResolver{err: expected}

	_, err := m.ResolveBaseURL(context.Background(), "any-id")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expected.Error() {
		t.Fatalf("expected error %q, got %q", expected.Error(), err.Error())
	}
}

func TestDBResolver_NilQueries(t *testing.T) {
	_, err := NewDBResolver(nil)
	if err == nil {
		t.Fatal("expected error for nil queries, got nil")
	}
}
