package main

import "testing"

func TestParseUIDs_Basic(t *testing.T) {
	got, err := parseUIDs("35385,35392,35396")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []uint32{35385, 35392, 35396}
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("uid[%d]=%d, want %d", i, got[i], want[i])
		}
	}
}

// Espaces autour des virgules et valeurs vides (double virgule) ignorées proprement.
func TestParseUIDs_TrimsAndSkipsEmpty(t *testing.T) {
	got, err := parseUIDs(" 1, 2 ,,3 ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []uint32{1, 2, 3}
	if len(got) != len(want) {
		t.Fatalf("len=%d, want %d (%v)", len(got), len(want), got)
	}
}

func TestParseUIDs_EmptyString(t *testing.T) {
	got, err := parseUIDs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("attendu slice vide, got %v", got)
	}
}

func TestParseUIDs_InvalidValue(t *testing.T) {
	_, err := parseUIDs("35385,abc,35396")
	if err == nil {
		t.Fatal("attendu une erreur sur une valeur non-numérique")
	}
}

func TestTruncate_ShortStringUnchanged(t *testing.T) {
	if got := truncate("short", 60); got != "short" {
		t.Errorf("got %q", got)
	}
}

func TestTruncate_LongStringCutWithEllipsis(t *testing.T) {
	s := "0123456789"
	got := truncate(s, 5)
	if got != "01234…" {
		t.Errorf("got %q", got)
	}
}
