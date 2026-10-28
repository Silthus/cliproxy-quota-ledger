package ledger_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/Silthus/cliproxy-quota-ledger/internal/ledger"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func reading(at time.Time, remaining float64) ledger.Sample {
	return ledger.Sample{
		At:        at.UnixMilli(),
		Auth:      "claude-team.json",
		Provider:  "claude",
		Window:    "five_hour",
		Remaining: remaining,
	}
}

func ms(value int64) *int64 { return &value }

func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*ledger.Sample)
		wantErr string
	}{
		{name: "valid", mutate: func(*ledger.Sample) {}},
		{name: "valid with resets_at", mutate: func(s *ledger.Sample) { s.ResetsAt = ms(t0.Add(time.Hour).UnixMilli()) }},
		{name: "at exactly 10 minutes ahead", mutate: func(s *ledger.Sample) { s.At = t0.Add(10 * time.Minute).UnixMilli() }},
		{name: "256 multi-byte auth characters", mutate: func(s *ledger.Sample) { s.Auth = strings.Repeat("é", 256) }},
		{name: "remaining bounds", mutate: func(s *ledger.Sample) { s.Remaining = 100 }},
		{name: "missing at", mutate: func(s *ledger.Sample) { s.At = 0 }, wantErr: "at must be"},
		{name: "at too far ahead", mutate: func(s *ledger.Sample) { s.At = t0.Add(10*time.Minute + time.Millisecond).UnixMilli() }, wantErr: "future"},
		{name: "empty auth", mutate: func(s *ledger.Sample) { s.Auth = "" }, wantErr: "auth must be 1..256"},
		{name: "long auth", mutate: func(s *ledger.Sample) { s.Auth = strings.Repeat("a", 257) }, wantErr: "auth must be 1..256"},
		{name: "long provider", mutate: func(s *ledger.Sample) { s.Provider = strings.Repeat("p", 65) }, wantErr: "provider must be 1..64"},
		{name: "empty window", mutate: func(s *ledger.Sample) { s.Window = "" }, wantErr: "window must be 1..128"},
		{name: "long window", mutate: func(s *ledger.Sample) { s.Window = strings.Repeat("w", 129) }, wantErr: "window must be 1..128"},
		{name: "negative remaining", mutate: func(s *ledger.Sample) { s.Remaining = -0.1 }, wantErr: "remaining must be"},
		{name: "remaining above 100", mutate: func(s *ledger.Sample) { s.Remaining = 100.1 }, wantErr: "remaining must be"},
		{name: "non-positive resets_at", mutate: func(s *ledger.Sample) { s.ResetsAt = ms(0) }, wantErr: "resets_at must be"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sample := reading(t0, 42)
			tc.mutate(&sample)

			err := sample.Validate(t0)

			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("Validate() = %v, want error containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestSampleJSON(t *testing.T) {
	t.Run("remaining is required", func(t *testing.T) {
		var sample ledger.Sample
		err := json.Unmarshal([]byte(`{"at":1,"auth":"a","provider":"p","window":"w"}`), &sample)
		if err == nil || !strings.Contains(err.Error(), "remaining is required") {
			t.Fatalf("Unmarshal() = %v, want remaining is required", err)
		}
	})

	t.Run("resets_at may be null or absent", func(t *testing.T) {
		for _, raw := range []string{
			`{"at":1,"auth":"a","provider":"p","window":"w","remaining":0}`,
			`{"at":1,"auth":"a","provider":"p","window":"w","remaining":0,"resets_at":null}`,
		} {
			var sample ledger.Sample
			if err := json.Unmarshal([]byte(raw), &sample); err != nil {
				t.Fatalf("Unmarshal(%s) = %v", raw, err)
			}
			if sample.ResetsAt != nil || sample.Remaining != 0 {
				t.Fatalf("Unmarshal(%s) = %+v", raw, sample)
			}
		}
	})

	t.Run("round trips", func(t *testing.T) {
		original := reading(t0, 12.5)
		original.ResetsAt = ms(t0.Add(time.Hour).UnixMilli())
		raw, err := json.Marshal(original)
		if err != nil {
			t.Fatal(err)
		}
		var decoded ledger.Sample
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatal(err)
		}
		if decoded.At != original.At || decoded.Remaining != original.Remaining || *decoded.ResetsAt != *original.ResetsAt {
			t.Fatalf("round trip = %+v, want %+v", decoded, original)
		}
	})
}
