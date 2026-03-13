// Tests for update_check.go — UpdateCheckResponse custom UnmarshalJSON handling.
package models

import (
	"encoding/json"
	"testing"
)

func TestUpdateCheckResponse_UnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		input       string
		wantFilm    int
		wantSorozat int
		wantErr     bool
	}{
		{
			name:        "integer values",
			input:       `{"film":5,"sorozat":10}`,
			wantFilm:    5,
			wantSorozat: 10,
		},
		{
			name:        "string values",
			input:       `{"film":"3","sorozat":"7"}`,
			wantFilm:    3,
			wantSorozat: 7,
		},
		{
			name:        "nil values",
			input:       `{"film":null,"sorozat":null}`,
			wantFilm:    0,
			wantSorozat: 0,
		},
		{
			name:        "mixed int film and string sorozat",
			input:       `{"film":12,"sorozat":"8"}`,
			wantFilm:    12,
			wantSorozat: 8,
		},
		{
			name:        "mixed string film and int sorozat",
			input:       `{"film":"4","sorozat":20}`,
			wantFilm:    4,
			wantSorozat: 20,
		},
		{
			name:        "zero integer values",
			input:       `{"film":0,"sorozat":0}`,
			wantFilm:    0,
			wantSorozat: 0,
		},
		{
			name:        "zero string values",
			input:       `{"film":"0","sorozat":"0"}`,
			wantFilm:    0,
			wantSorozat: 0,
		},
		{
			name:    "invalid JSON",
			input:   `{not json}`,
			wantErr: true,
		},
		{
			name:    "invalid film string",
			input:   `{"film":"abc","sorozat":"1"}`,
			wantErr: true,
		},
		{
			name:    "invalid sorozat string",
			input:   `{"film":"1","sorozat":"xyz"}`,
			wantErr: true,
		},
		{
			name:    "unsupported type for film",
			input:   `{"film":true,"sorozat":1}`,
			wantErr: true,
		},
		{
			name:    "unsupported type for sorozat",
			input:   `{"film":1,"sorozat":[1,2]}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var resp UpdateCheckResponse
			err := json.Unmarshal([]byte(tt.input), &resp)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("UnmarshalJSON(%s) expected error, got nil", tt.input)
				}
				return
			}

			if err != nil {
				t.Fatalf("UnmarshalJSON(%s) unexpected error: %v", tt.input, err)
			}

			if resp.Film != tt.wantFilm {
				t.Errorf("Film = %d, want %d", resp.Film, tt.wantFilm)
			}
			if resp.Sorozat != tt.wantSorozat {
				t.Errorf("Sorozat = %d, want %d", resp.Sorozat, tt.wantSorozat)
			}
		})
	}
}

func TestUpdateCheckResponse_UnmarshalJSON_MissingFields(t *testing.T) {
	t.Parallel()
	var resp UpdateCheckResponse
	err := json.Unmarshal([]byte(`{}`), &resp)
	if err != nil {
		t.Fatalf("UnmarshalJSON({}) unexpected error: %v", err)
	}
	if resp.Film != 0 {
		t.Errorf("Film = %d, want 0 for missing field", resp.Film)
	}
	if resp.Sorozat != 0 {
		t.Errorf("Sorozat = %d, want 0 for missing field", resp.Sorozat)
	}
}

func TestUpdateCheckResult_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	original := UpdateCheckResult{
		FilmCount:   5,
		SeriesCount: 10,
		HasUpdates:  true,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() unexpected error: %v", err)
	}

	var decoded UpdateCheckResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() unexpected error: %v", err)
	}

	if decoded.FilmCount != original.FilmCount {
		t.Errorf("FilmCount = %d, want %d", decoded.FilmCount, original.FilmCount)
	}
	if decoded.SeriesCount != original.SeriesCount {
		t.Errorf("SeriesCount = %d, want %d", decoded.SeriesCount, original.SeriesCount)
	}
	if decoded.HasUpdates != original.HasUpdates {
		t.Errorf("HasUpdates = %v, want %v", decoded.HasUpdates, original.HasUpdates)
	}
}
