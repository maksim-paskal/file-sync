package main

import "testing"

func TestNewSHA256(t *testing.T) {
	got := NewSHA256([]byte("dsdd"))
	if got != "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5" {
		t.Error("SHA256 is not correct")
	}
}
