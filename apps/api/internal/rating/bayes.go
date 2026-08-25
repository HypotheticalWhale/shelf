// Package rating implements Shelf's aggregate scoring.
package rating

import (
	"errors"
	"math"
)

const (
	// Min and Max bound the 1..10 scale board gamers already know from BGG.
	Min = 1.0
	Max = 10.0
	// Step is the granularity of a rating. Half-points give people enough
	// resolution to separate games without the false precision of decimals.
	Step = 0.5
)

// ErrOutOfRange is returned by Validate for values off the 1..10 half-point scale.
var ErrOutOfRange = errors.New("rating must be between 1 and 10 in steps of 0.5")

// Validate reports whether v is a legal rating.
func Validate(v float64) error {
	if math.IsNaN(v) || v < Min || v > Max {
		return ErrOutOfRange
	}
	// Guard against float noise: 7.5/0.5 may land a hair off an integer.
	steps := v / Step
	if math.Abs(steps-math.Round(steps)) > 1e-9 {
		return ErrOutOfRange
	}
	return nil
}

// Score returns the Bayesian weighted average for one game.
//
//	score = (sum + m*C) / (n + m)
//
// where sum and n describe the game's own ratings, C is the global mean across
// every rating on the site, and m is the prior weight — the number of imaginary
// "perfectly average" votes each game is seeded with.
//
// The plain mean is the wrong tool here: a single 10/10 would rank an unknown
// game above Gloomhaven. The prior pulls sparsely-rated games toward C and
// releases them as real votes accumulate, so the leaderboard reflects both how
// good a game is and how many people actually vouched for it.
//
// With n == 0 the result is exactly C, so an unrated game sits mid-pack rather
// than at zero.
func Score(sum float64, n int, globalMean float64, priorWeight int) float64 {
	if n < 0 {
		n = 0
	}
	if priorWeight < 0 {
		priorWeight = 0
	}
	denom := float64(n + priorWeight)
	if denom == 0 {
		// No ratings and no prior: nothing to say about this game.
		return globalMean
	}
	return (sum + float64(priorWeight)*globalMean) / denom
}

// Mean returns the unweighted average, or 0 when a game has no ratings. Shown
// alongside Score on game pages so people can see the raw crowd opinion next to
// the ranked figure.
func Mean(sum float64, n int) float64 {
	if n <= 0 {
		return 0
	}
	return sum / float64(n)
}
