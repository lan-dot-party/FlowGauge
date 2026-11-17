package api

import (
	"net/url"
	"testing"
)

func TestParsePaging(t *testing.T) {
	tests := []struct {
		query      string
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{"", defaultResultsLimit, 0, false},
		{"limit=50&offset=10", 50, 10, false},
		{"limit=200", 200, 0, false},
		{"limit=99999999", maxResultsLimit, 0, false},
		{"limit=abc", 0, 0, true},
		{"limit=0", 0, 0, true},
		{"limit=-5", 0, 0, true},
		{"offset=-1", 0, 0, true},
		{"offset=x", 0, 0, true},
	}
	for _, tt := range tests {
		q, _ := url.ParseQuery(tt.query)
		limit, offset, err := parsePaging(q)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q: err = %v, wantErr %v", tt.query, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && (limit != tt.wantLimit || offset != tt.wantOffset) {
			t.Errorf("%q: got limit=%d offset=%d, want %d/%d", tt.query, limit, offset, tt.wantLimit, tt.wantOffset)
		}
	}
}
