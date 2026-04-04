package domain

import (
	"math"
	"testing"
	"time"
)

func makeParticipants(votes map[string]VoteValue) map[string]*Participant {
	result := make(map[string]*Participant)
	i := 0
	for name, vote := range votes {
		sid := "s" + name
		result[sid] = &Participant{
			SessionID: sid,
			Name:      name,
			Vote:      vote,
			Status:    "active",
			LastPing:  time.Now(),
		}
		i++
	}
	return result
}

func floatEq(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return math.Abs(*a-*b) < 0.01
}

func fptr(v float64) *float64 { return &v }

func TestCalculateResult(t *testing.T) {
	tests := []struct {
		name           string
		votes          map[string]VoteValue
		wantAvg        *float64
		wantMedian     *float64
		wantUncertain  int
		wantTotal      int
		wantConsensus  bool
		wantSpread     *[2]float64
	}{
		{
			name:          "all numeric, consensus",
			votes:         map[string]VoteValue{"A": "5", "B": "5", "C": "5"},
			wantAvg:       fptr(5),
			wantMedian:    fptr(5),
			wantUncertain: 0,
			wantTotal:     3,
			wantConsensus: true,
			wantSpread:    &[2]float64{5, 5},
		},
		{
			name:          "mixed numeric",
			votes:         map[string]VoteValue{"A": "1", "B": "3", "C": "8"},
			wantAvg:       fptr(4),
			wantMedian:    fptr(3),
			wantUncertain: 0,
			wantTotal:     3,
			wantConsensus: false,
			wantSpread:    &[2]float64{1, 8},
		},
		{
			name:          "with uncertainty",
			votes:         map[string]VoteValue{"A": "5", "B": "?", "C": "8"},
			wantAvg:       fptr(6.5),
			wantMedian:    fptr(6.5),
			wantUncertain: 1,
			wantTotal:     3,
			wantConsensus: false,
			wantSpread:    &[2]float64{5, 8},
		},
		{
			name:          "all uncertain",
			votes:         map[string]VoteValue{"A": "?", "B": "?"},
			wantAvg:       nil,
			wantMedian:    nil,
			wantUncertain: 2,
			wantTotal:     2,
			wantConsensus: false,
			wantSpread:    nil,
		},
		{
			name:          "no votes at all",
			votes:         map[string]VoteValue{"A": "", "B": ""},
			wantAvg:       nil,
			wantMedian:    nil,
			wantUncertain: 0,
			wantTotal:     0,
			wantConsensus: false,
			wantSpread:    nil,
		},
		{
			name:          "single voter",
			votes:         map[string]VoteValue{"A": "13"},
			wantAvg:       fptr(13),
			wantMedian:    fptr(13),
			wantUncertain: 0,
			wantTotal:     1,
			wantConsensus: true,
			wantSpread:    &[2]float64{13, 13},
		},
		{
			name:          "even number of numeric votes",
			votes:         map[string]VoteValue{"A": "2", "B": "3", "C": "5", "D": "8"},
			wantAvg:       fptr(4.5),
			wantMedian:    fptr(4),
			wantUncertain: 0,
			wantTotal:     4,
			wantConsensus: false,
			wantSpread:    &[2]float64{2, 8},
		},
		{
			name:          "zero votes",
			votes:         map[string]VoteValue{"A": "0", "B": "0"},
			wantAvg:       fptr(0),
			wantMedian:    fptr(0),
			wantUncertain: 0,
			wantTotal:     2,
			wantConsensus: true,
			wantSpread:    &[2]float64{0, 0},
		},
		{
			name:          "half point votes",
			votes:         map[string]VoteValue{"A": "0.5", "B": "0.5"},
			wantAvg:       fptr(0.5),
			wantMedian:    fptr(0.5),
			wantUncertain: 0,
			wantTotal:     2,
			wantConsensus: true,
			wantSpread:    &[2]float64{0.5, 0.5},
		},
		{
			name:          "empty participants map",
			votes:         map[string]VoteValue{},
			wantAvg:       nil,
			wantMedian:    nil,
			wantUncertain: 0,
			wantTotal:     0,
			wantConsensus: false,
			wantSpread:    nil,
		},
		{
			name:          "mixed empty and voted",
			votes:         map[string]VoteValue{"A": "3", "B": "", "C": "5"},
			wantAvg:       fptr(4),
			wantMedian:    fptr(4),
			wantUncertain: 0,
			wantTotal:     2,
			wantConsensus: false,
			wantSpread:    &[2]float64{3, 5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			participants := makeParticipants(tt.votes)
			result := CalculateResult(participants)

			if !floatEq(result.Average, tt.wantAvg) {
				t.Errorf("Average = %v, want %v", ptrStr(result.Average), ptrStr(tt.wantAvg))
			}
			if !floatEq(result.Median, tt.wantMedian) {
				t.Errorf("Median = %v, want %v", ptrStr(result.Median), ptrStr(tt.wantMedian))
			}
			if result.UncertainCount != tt.wantUncertain {
				t.Errorf("UncertainCount = %d, want %d", result.UncertainCount, tt.wantUncertain)
			}
			if result.TotalVoters != tt.wantTotal {
				t.Errorf("TotalVoters = %d, want %d", result.TotalVoters, tt.wantTotal)
			}
			if result.HasConsensus != tt.wantConsensus {
				t.Errorf("HasConsensus = %v, want %v", result.HasConsensus, tt.wantConsensus)
			}
			if tt.wantSpread == nil && result.Spread != nil {
				t.Errorf("Spread = %v, want nil", result.Spread)
			}
			if tt.wantSpread != nil {
				if result.Spread == nil {
					t.Errorf("Spread = nil, want %v", tt.wantSpread)
				} else if result.Spread[0] != tt.wantSpread[0] || result.Spread[1] != tt.wantSpread[1] {
					t.Errorf("Spread = %v, want %v", result.Spread, tt.wantSpread)
				}
			}
		})
	}
}

func ptrStr(p *float64) string {
	if p == nil {
		return "<nil>"
	}
	return ""
}
