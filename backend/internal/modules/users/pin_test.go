package users

import "testing"

func TestValidPIN(t *testing.T) {
	good := []string{"4821", "90731", "250819", "1357"}
	bad := []string{"123", "1234567", "1234", "4321", "0000", "56789", "12a4", ""}
	for _, p := range good {
		if !validPIN(p) {
			t.Errorf("validPIN(%q) = false, want true", p)
		}
	}
	for _, p := range bad {
		if validPIN(p) {
			t.Errorf("validPIN(%q) = true, want false", p)
		}
	}
}
