package adapters

import (
	"testing"
)

func TestNewAdapter(t *testing.T) {
	tests := []struct {
		connString string
		wantErr    bool
	}{
		{"postgres://user:pass@localhost:5432/db", false},
		{"postgresql://user:pass@localhost:5432/db", false},
		{"mysql://user:pass@localhost:3306/db", true}, // Not implemented yet
		{"sqlite://db.sqlite", true},                  // Not implemented yet
		{"invalid-scheme", true},
	}

	for _, tt := range tests {
		adapter, err := NewAdapter(tt.connString)
		if (err != nil) != tt.wantErr {
			t.Errorf("NewAdapter(%q) error = %v, wantErr %v", tt.connString, err, tt.wantErr)
		}
		if !tt.wantErr && adapter == nil {
			t.Errorf("NewAdapter(%q) returned nil adapter", tt.connString)
		}
	}
}
