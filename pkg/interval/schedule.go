package interval

import "cmp"

// A group contains repeats of the same "thing" at
// different intervals.
type Group[K comparable, T cmp.Ordered] struct {
	Key       K
	Intervals []Interval[T]
}

// A schedule is an ordered set of {interval, key} pairs,
// such that no key is present more than once and no
// intervals overlap.
type Schedule[K comparable, T cmp.Ordered] []ScheduleEntry[K, T]

type ScheduleEntry[K comparable, T cmp.Ordered] struct {
	Interval[T]
	Key K
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
