package interval

import (
	"cmp"
	"slices"
)

// A group contains repeats of the same "thing" at
// different intervals.
type Group[K comparable, T cmp.Ordered] struct {
	Key       K
	Intervals []Interval[T]
}

// A schedule is a set of {interval, key} pairs,
// such that no key is present more than once and no
// intervals overlap.
type Schedule[K comparable, T cmp.Ordered] map[K]Interval[T]

// Equals checks if two schedules contain the same entries.
func (s Schedule[K, T]) Equals(other Schedule[K, T]) bool {
	if len(s) != len(other) {
		return false
	}
	for k, v := range s {
		vOther, ok := other[k]
		if !ok || v != vOther {
			return false
		}
	}
	return true
}

// Get the groups in order of first to last. You can
// use the result to then retrieve the intervals.
func (s Schedule[K, T]) OrderedKeys() []K {
	// Convert to pair list
	type pair struct {
		key      K
		interval Interval[T]
	}
	pairs := make([]pair, len(s))
	i := 0
	for key, interval := range s {
		pairs[i].key = key
		pairs[i].interval = interval
		i++
	}

	// Sort pairs by the interval
	slices.SortFunc(pairs, func(a, b pair) int {
		return Cmp(a.interval, b.interval)
	})

	// Return the keys
	keys := make([]K, len(pairs))
	for i, p := range pairs {
		keys[i] = p.key
	}
	return keys
}

// This is the GISMP variant of the interval scheduling problem.
// Given a set of groups with their intervals, come up with
// "maximal" schedules. Given two schedules, one of the schedules
// is maximal with respect to the other if it is a superset of
// the other. This function can return multiple maximal schedules,
// since some maximal schedules may have uncommon elements.
func GroupIntervalSchedulingMaximization[K comparable, T cmp.Ordered](groups []Group[K, T]) []Schedule[K, T] {
	// TODO
	return nil
}
