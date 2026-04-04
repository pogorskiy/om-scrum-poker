package domain

import (
	"math"
	"sort"
	"strconv"
)

// VoteEntry holds a single participant's revealed vote.
type VoteEntry struct {
	SessionID string  `json:"sessionId"`
	Name      string  `json:"name"`
	Value     string  `json:"value"`
}

// RoundResult contains aggregated voting statistics.
type RoundResult struct {
	Votes          []VoteEntry `json:"votes"`
	Average        *float64    `json:"average"`
	Median         *float64    `json:"median"`
	UncertainCount int         `json:"uncertainCount"`
	TotalVoters    int         `json:"totalVoters"`
	HasConsensus   bool        `json:"hasConsensus"`
	Spread         *[2]float64 `json:"spread"`
}

// CalculateResult computes statistics from the participants' votes.
// Only numeric votes contribute to average, median, consensus, and spread.
// "?" votes count as uncertain. Empty votes are excluded entirely.
func CalculateResult(participants map[string]*Participant) *RoundResult {
	result := &RoundResult{
		Votes: make([]VoteEntry, 0),
	}

	var numericValues []float64

	for _, p := range participants {
		if p.Vote == "" {
			continue
		}
		result.TotalVoters++

		result.Votes = append(result.Votes, VoteEntry{
			SessionID: p.SessionID,
			Name:      p.Name,
			Value:     string(p.Vote),
		})

		if p.Vote == "?" {
			result.UncertainCount++
			continue
		}

		val, err := strconv.ParseFloat(string(p.Vote), 64)
		if err != nil {
			continue
		}
		numericValues = append(numericValues, val)
	}

	// Sort votes by session ID for deterministic output.
	sort.Slice(result.Votes, func(i, j int) bool {
		return result.Votes[i].SessionID < result.Votes[j].SessionID
	})

	if len(numericValues) == 0 {
		return result
	}

	sort.Float64s(numericValues)

	// Average
	avg := average(numericValues)
	result.Average = &avg

	// Median
	med := median(numericValues)
	result.Median = &med

	// Consensus: all numeric votes are identical.
	result.HasConsensus = isConsensus(numericValues)

	// Spread: [min, max]
	spread := [2]float64{numericValues[0], numericValues[len(numericValues)-1]}
	result.Spread = &spread

	return result
}

func average(values []float64) float64 {
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	avg := sum / float64(len(values))
	return math.Round(avg*100) / 100
}

func median(values []float64) float64 {
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2
	}
	return values[n/2]
}

func isConsensus(values []float64) bool {
	if len(values) <= 1 {
		return len(values) == 1
	}
	first := values[0]
	for _, v := range values[1:] {
		if v != first {
			return false
		}
	}
	return true
}
