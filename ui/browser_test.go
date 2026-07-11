package ui

import "testing"

func TestShouldComputeVisibleRange(t *testing.T) {
	tests := []struct {
		name                 string
		cursor, total, rows  int
		wantStart, wantEnd   int
	}{
		{"fits entirely", 0, 5, 20, 0, 5},
		{"top of long list", 0, 100, 20, 0, 20},
		{"cursor mid-list is inside window", 50, 100, 20, 31, 51},
		{"cursor at end", 99, 100, 20, 80, 100},
		{"single row viewport", 7, 100, 1, 7, 8},
		{"empty list", 0, 0, 20, 0, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := visibleRange(tt.cursor, tt.total, tt.rows)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Fatalf("visibleRange(%d,%d,%d) = (%d,%d), want (%d,%d)",
					tt.cursor, tt.total, tt.rows, start, end, tt.wantStart, tt.wantEnd)
			}
			if tt.total > 0 && (tt.cursor < start || tt.cursor >= end) {
				t.Fatalf("cursor %d outside visible window [%d,%d)", tt.cursor, start, end)
			}
		})
	}
}
