package rating

import (
	"math"
	"testing"
)

const eps = 1e-9

func TestScoreUnratedGameSitsAtGlobalMean(t *testing.T) {
	// The defining property of the prior: with no votes, a game is indis-
	// tinguishable from average rather than ranked at zero.
	got := Score(0, 0, 7.2, 5)
	if math.Abs(got-7.2) > eps {
		t.Fatalf("Score(0,0,7.2,5) = %v, want 7.2", got)
	}
}

func TestScoreSingleHighRatingCannotTopTheChart(t *testing.T) {
	// This is the failure mode a plain mean has, and the whole reason the
	// Bayesian average exists: one enthusiastic vote must not outrank a
	// broadly-loved game.
	newcomer := Score(10, 1, 7.0, 5) // a lone 10/10
	classic := Score(8.5*200, 200, 7.0, 5)

	if newcomer >= classic {
		t.Fatalf("a single 10/10 (%v) outranked a 200-vote 8.5 average (%v)", newcomer, classic)
	}
	if math.Abs(newcomer-7.5) > eps {
		t.Fatalf("Score(10,1,7,5) = %v, want 7.5", newcomer)
	}
}

func TestScoreConvergesToTrueMeanAsVotesGrow(t *testing.T) {
	// The prior must fade, not persist — with enough votes the score should
	// approach the game's own average.
	const trueMean, globalMean = 9.0, 7.0
	prev := math.Inf(-1)

	for _, n := range []int{1, 10, 100, 1000, 100000} {
		got := Score(trueMean*float64(n), n, globalMean, 5)
		if got <= prev {
			t.Fatalf("score not increasing toward true mean at n=%d: %v then %v", n, prev, got)
		}
		if got > trueMean {
			t.Fatalf("score %v at n=%d overshot the true mean %v", got, n, trueMean)
		}
		prev = got
	}
	if math.Abs(prev-trueMean) > 0.01 {
		t.Fatalf("with 100k votes score = %v, want ~%v", prev, trueMean)
	}
}

func TestScoreDegenerateInputs(t *testing.T) {
	// A zero prior means "just use the mean"; a zero prior AND zero votes has
	// no defined answer, so fall back to the global mean instead of dividing
	// by zero.
	if got := Score(0, 0, 6.5, 0); math.Abs(got-6.5) > eps {
		t.Fatalf("Score with no votes and no prior = %v, want global mean 6.5", got)
	}
	if got := Score(16, 2, 7.0, 0); math.Abs(got-8.0) > eps {
		t.Fatalf("Score with zero prior = %v, want the plain mean 8.0", got)
	}
	if got := Score(0, -3, 7.0, -2); math.Abs(got-7.0) > eps {
		t.Fatalf("negative inputs should clamp to the global mean, got %v", got)
	}
	if math.IsNaN(Score(0, 0, 7.0, 0)) {
		t.Fatal("Score returned NaN")
	}
}

func TestMean(t *testing.T) {
	if got := Mean(0, 0); got != 0 {
		t.Fatalf("Mean(0,0) = %v, want 0", got)
	}
	if got := Mean(21, 3); math.Abs(got-7) > eps {
		t.Fatalf("Mean(21,3) = %v, want 7", got)
	}
}

func TestValidate(t *testing.T) {
	valid := []float64{1, 1.5, 5, 7.5, 9.5, 10}
	for _, v := range valid {
		if err := Validate(v); err != nil {
			t.Errorf("Validate(%v) = %v, want nil", v, err)
		}
	}

	invalid := []float64{0, 0.5, 10.5, 11, -1, 7.3, math.NaN()}
	for _, v := range invalid {
		if err := Validate(v); err == nil {
			t.Errorf("Validate(%v) = nil, want an error", v)
		}
	}
}
