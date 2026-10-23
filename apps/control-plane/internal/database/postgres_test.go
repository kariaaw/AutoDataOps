package database

import (
	"context"
	"testing"
)

func TestOpenRejectsInvalidDSN(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"malformed", "postgres://%zz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pool, err := Open(context.Background(), tt.dsn)

			if pool != nil {
				pool.Close()
				t.Fatal("expected no connection pool")
			}

			if err == nil {
				t.Fatal("expected a configuration error")
			}
		})
	}
}
