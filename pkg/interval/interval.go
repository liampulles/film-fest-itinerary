package interval

import (
	"cmp"
	"errors"
)

var (
	ErrInvalidInterval = errors.New("invalid interval; end may not precede the start")
)

// An Interval has a start and an end, where the end may not
// precede the start.
//
// We consider our intervals to have a "closed" start and an "open"
// end. This is so that two intervals that are "touching" start
// to end can be said to be non-overlapping.
//
// For ease of use, we treat Intervals as immutable.
//
// For using times and intervals, simply use the unix epoch
// values.
//
// Struct equality is sufficient to test Interval equality.
type Interval[T cmp.Ordered] struct {
	start T
	end   T
}

// Create a new Interval. Returns an error if validation fails.
func New[T cmp.Ordered](start, end T) (Interval[T], error) {
	if end < start {
		return Interval[T]{}, ErrInvalidInterval
	}

	return Interval[T]{
		start: start,
		end:   end,
	}, nil
}

// Start returns the interval's start value.
func (a Interval[T]) Start() T {
	return a.start
}

// End returns the interval's end value.
func (a Interval[T]) End() T {
	return a.end
}

func (a Interval[T]) Before(b Interval[T]) bool {
	return a.start < b.start
}

func (a Interval[T]) After(b Interval[T]) bool {
	return a.start > b.start
}

func (a Interval[T]) Touches(b Interval[T]) bool {
	return a.end == b.start || b.end == a.start
}

func (a Interval[T]) Overlaps(b Interval[T]) bool {
	return a.start < b.end && b.start < a.end
}

// Can be used for sorting
func Cmp[T cmp.Ordered](i, j Interval[T]) int {
	if i.start != j.start {
		return cmp.Compare(i.start, j.start)
	}
	return cmp.Compare(i.end, j.end)
}

// The relationship between two intervals is basically
// how they overlap or else the order. Given intervals
// a and b, you can read the different
// relationships as "a <Relationship> b"
type Relationship int

const (
	StrictlyBefore Relationship = iota
	TouchingBefore
	PartialOverlapBefore
	ContainsTouchingEnd
	ContainsInMiddle
	ContainedWithinTouchingStart
	Equals
	ContainsTouchingStart
	ContainedWithinInMiddle
	ContainedWithinTouchingEnd
	PartialOverlapAfter
	TouchingAfter
	StrictlyAfter
)

func (a Interval[T]) Relationship(b Interval[T]) Relationship {
	switch {
	case a.start < b.start:
		switch {
		case a.end < b.start:
			return StrictlyBefore
		case a.end == b.start:
			return TouchingBefore
		case a.end > b.end:
			return ContainsInMiddle
		case a.end == b.end:
			return ContainsTouchingEnd
		// a.end > b.start AND a.end < b.end
		default:
			return PartialOverlapBefore
		}

	case a.start == b.start:
		switch {
		case a.end < b.end:
			return ContainedWithinTouchingStart
		case a.end > b.end:
			return ContainsTouchingStart
		// a.end == b.end
		default:
			return Equals
		}

	// a.start > b.start
	default:
		switch {
		case a.end < b.end:
			return ContainedWithinInMiddle
		case a.end == b.end:
			return ContainedWithinTouchingEnd
		case a.start > b.end:
			return StrictlyAfter
		case a.start == b.end:
			return TouchingAfter
		// a.start < b.end AND a.end > b.end
		default:
			return PartialOverlapAfter
		}
	}
}
