package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiscoverAddedAndRemovedFeature(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "features", "Fun_hello")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "feature.go"), []byte("package hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(filepath.Join(root, "features", "catalog_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "personal-discord-bot/features/Fun_hello") {
		t.Fatal("feature not discovered")
	}
	if err := os.RemoveAll(dir); err != nil {
		t.Fatal(err)
	}
	if err := generate(root); err != nil {
		t.Fatal(err)
	}
	out, err = os.ReadFile(filepath.Join(root, "features", "catalog_generated.go"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(out), "features/Fun_hello") {
		t.Fatal("deleted feature still imported")
	}
}
