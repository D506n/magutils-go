package id

import (
	"testing"
)

func TestGen_Default(t *testing.T) {
	id, err := Gen()
	if err != nil {
		t.Fatalf("Gen() returned error: %v", err)
	}
	if len(id) != DefaultSize {
		t.Errorf("expected length %d, got %d", DefaultSize, len(id))
	}
}

func TestGen_Unique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for i := 0; i < 100; i++ {
		id, err := Gen()
		if err != nil {
			t.Fatalf("Gen() returned error: %v", err)
		}
		if _, ok := seen[id]; ok {
			t.Errorf("duplicate id generated: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestGen_Alphabet(t *testing.T) {
	// Только hex
	err := SetAlphabet("0123456789abcdef")
	if err != nil {
		t.Fatalf("SetAlphabet() returned error: %v", err)
	}
	defer SetAlphabet(DefaultAlphabet) // восстановим

	id, err := Gen()
	if err != nil {
		t.Fatalf("Gen() returned error: %v", err)
	}
	for _, ch := range id {
		if !((ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f')) {
			t.Errorf("unexpected character %c in hex id", ch)
		}
	}
}

func TestGenWith_CustomSize(t *testing.T) {
	id, err := GenWith(DefaultAlphabet, 10)
	if err != nil {
		t.Fatalf("GenWith() returned error: %v", err)
	}
	if len(id) != 10 {
		t.Errorf("expected length 10, got %d", len(id))
	}
}

func TestGenWith_InvalidSize(t *testing.T) {
	_, err := GenWith(DefaultAlphabet, 3)
	if err == nil {
		t.Error("expected error for size < 5, got nil")
	}
}

func TestGenWith_InvalidAlphabet(t *testing.T) {
	_, err := GenWith("abc", DefaultSize)
	if err == nil {
		t.Error("expected error for alphabet < 10 chars, got nil")
	}
}

func TestMustGen(t *testing.T) {
	id := MustGen()
	if len(id) != DefaultSize {
		t.Errorf("expected length %d, got %d", DefaultSize, len(id))
	}
}

func TestSetSize(t *testing.T) {
	defer SetSize(DefaultSize)

	err := SetSize(10)
	if err != nil {
		t.Fatalf("SetSize(10) returned error: %v", err)
	}
	if Size() != 10 {
		t.Errorf("expected Size()=10, got %d", Size())
	}

	err = SetSize(3)
	if err == nil {
		t.Error("expected error for size < 5, got nil")
	}
}

func TestSetAlphabet(t *testing.T) {
	defer SetAlphabet(DefaultAlphabet)

	err := SetAlphabet("0123456789abcdef")
	if err != nil {
		t.Fatalf("SetAlphabet() returned error: %v", err)
	}
	if Alphabet() != "0123456789abcdef" {
		t.Errorf("expected hex alphabet, got %q", Alphabet())
	}

	err = SetAlphabet("abc")
	if err == nil {
		t.Error("expected error for alphabet < 10 chars, got nil")
	}

	err = SetAlphabet("aabbccddee")
	if err == nil {
		t.Error("expected error for non-unique alphabet, got nil")
	}
}

func TestIsValid(t *testing.T) {
	id, err := Gen()
	if err != nil {
		t.Fatalf("Gen() returned error: %v", err)
	}

	if err := IsValid(id); err != nil {
		t.Errorf("IsValid(%q) returned error: %v", id, err)
	}

	if err := IsValid("short"); err == nil {
		t.Error("expected error for short id, got nil")
	}

	if err := IsValid("!!!!!!!!!!!!!!!!"); err == nil {
		t.Error("expected error for invalid chars, got nil")
	}
}

func TestAlphabetAndSize(t *testing.T) {
	if Alphabet() != DefaultAlphabet {
		t.Errorf("expected default alphabet, got %q", Alphabet())
	}
	if Size() != DefaultSize {
		t.Errorf("expected default size %d, got %d", DefaultSize, Size())
	}
}

func TestAllUnique(t *testing.T) {
	if !allUnique("abc") {
		t.Error("expected true for 'abc'")
	}
	if allUnique("aba") {
		t.Error("expected false for 'aba'")
	}
}
