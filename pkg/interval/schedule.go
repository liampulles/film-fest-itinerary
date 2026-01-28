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
		// If group is not present in other, or
		// the interval for the other is not the same,
		// then the schedules are not equal.
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
//
// This is not making any attempt to be the most efficient solution
// to the problem, which is known to be NP complete. Rather it is
// meant to be clear, stable, and somewhat reasonable in terms
// of algorithmic complexity.
func GroupIntervalSchedulingMaximization[K comparable, T cmp.Ordered](groups []Group[K, T]) []Schedule[K, T] {
	var results []Schedule[K, T]
	current := make(Schedule[K, T])

	var dfs func(idx int)
	dfs = func(idx int) {
		// We're at the last group - check if we've made a maximal group.
		// If so, add it to the set.
		if idx == len(groups) {
			if isMaximalSchedule(current, groups) {
				results = appendUniqueSchedule(results, cloneSchedule(current))
			}
			return
		}

		group := groups[idx]

		// If we've already got the working group in our WIP Schedule,
		// proceed to the next group.
		if _, exists := current[group.Key]; exists {
			dfs(idx + 1)
			return
		}

		// So this group is not in our schedule yet. Then see
		// if there is any gap where we could fit in one of its intervals.
		for _, interval := range group.Intervals {
			if !overlapsAny(interval, current) {
				// There is a gap. So...
				// -> Temporarily add it to the WIP schedule
				current[group.Key] = interval
				// -> Explore options where we include this interval
				dfs(idx + 1)
				// -> Remove this interval from the WIP schedule
				delete(current, group.Key)
				// -> Go to the next loop iteration. If there is
				//    a gap for another interval from this group,
				//    then we can explore options with it.
			}
		}

		dfs(idx + 1)
	}

	dfs(0)
	return results
}

func overlapsAny[K comparable, T cmp.Ordered](interval Interval[T], schedule Schedule[K, T]) bool {
	for _, existing := range schedule {
		if interval.Overlaps(existing) {
			return true
		}
	}
	return false
}

func isMaximalSchedule[K comparable, T cmp.Ordered](schedule Schedule[K, T], groups []Group[K, T]) bool {
	for _, group := range groups {
		// If we've already got the group in the schedule, then
		// we don't need to check it.
		if _, exists := schedule[group.Key]; exists {
			continue
		}

		// So the group is NOT in the schedule. Is there a
		// gap where one of this group's intervals could fit?
		for _, interval := range group.Intervals {
			if !overlapsAny(interval, schedule) {
				// There is a gap, which means this group could have
				// been included, which means this schedule
				// is not maximal (the schedule with this group
				// would be the maximal one).
				return false
			}
		}
	}
	return true
}

func cloneSchedule[K comparable, T cmp.Ordered](schedule Schedule[K, T]) Schedule[K, T] {
	clone := make(Schedule[K, T], len(schedule))
	for k, v := range schedule {
		clone[k] = v
	}
	return clone
}

func appendUniqueSchedule[K comparable, T cmp.Ordered](
	schedules []Schedule[K, T],
	candidate Schedule[K, T],
) []Schedule[K, T] {
	for _, existing := range schedules {
		if existing.Equals(candidate) {
			return schedules
		}
	}
	return append(schedules, candidate)
}
