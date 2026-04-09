// Copyright 2026 Alibaba Group
// Licensed under the Apache License, Version 2.0

package app

import (
	"context"
	"testing"
)

func TestResolveAuxiliaryAccessToken_explicitToken(t *testing.T) {
	tok, err := ResolveAuxiliaryAccessToken(context.Background(), "/any/dir", "  bearer-xyz  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok != "bearer-xyz" {
		t.Fatalf("got %q, want bearer-xyz", tok)
	}
}

func TestResolveAuxiliaryAccessToken_emptyConfigDir(t *testing.T) {
	_, err := ResolveAuxiliaryAccessToken(context.Background(), "  ", "")
	if err == nil {
		t.Fatal("expected error for empty config directory")
	}
}
