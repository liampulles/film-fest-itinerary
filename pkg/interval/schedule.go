package interval

import "cmp"

// A group contains repeats of the same "thing" at
// different intervals.
type Group[K comparable, T cmp.Ordered] struct {
	Key       K
	Intervals []Interval[T]
}

// A schedule is a set of {interval, key} pairs,
// such that no key is present more than once and no
// intervals overlap.
type Schedule[K comparable, T cmp.Ordered] []ScheduleEntry[K, T]

type ScheduleEntry[K comparable, T cmp.Ordered] struct {
	Interval[T]
	Key K
}

// Equals checks if two schedules contain the same entries, regardless of order.
func (s Schedule[K, T]) Equals(other Schedule[K, T]) bool {
	if len(s) != len(other) {
		return false
	}
	for _, entryA := range s {
		match := false
		for _, entryB := range other {
			if entryA.Key == entryB.Key && entryA.Interval == entryB.Interval {
				match = true
				break
			}
		}
		if !match {
			return false
		}
	}
	return true
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
