package interval

import (
	"testing"
)

func TestGroupIntervalSchedulingMaximization(t *testing.T) {
	// iv is a helper to create intervals without boilerplate.
	iv := func(start, end int) Interval[int] {
		i, err := New(start, end)
		if err != nil {
			t.Fatalf("New(%v, %v) failed: %v", start, end, err)
		}
		return i
	}

	tests := []struct {
		name   string
		groups []Group[string, int]
		want   [][]ScheduleEntry[string, int]
	}{
		{
			name: "No conflicts: Math and Science at different times",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(9, 10)}},
				{Key: "Science", Intervals: []Interval[int]{iv(11, 12)}},
			},
			want: [][]ScheduleEntry[string, int]{
				{
					{Key: "Math", Interval: iv(9, 10)},
					{Key: "Science", Interval: iv(11, 12)},
				},
			},
		},
		{
			name: "Simple conflict: Math and History overlap",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(9, 11)}},
				{Key: "History", Intervals: []Interval[int]{iv(10, 12)}},
			},
			want: [][]ScheduleEntry[string, int]{
				{{Key: "Math", Interval: iv(9, 11)}},
				{{Key: "History", Interval: iv(10, 12)}},
			},
		},
		{
			name: "Multiple options: Math has two slots, one overlaps with Science",
			groups: []Group[string, int]{
				{
					Key: "Math",
					Intervals: []Interval[int]{iv(9, 10), iv(14, 15)},
				},
				{
					Key: "Science",
					Intervals: []Interval[int]{iv(9, 10)},
				},
			},
			want: [][]ScheduleEntry[string, int]{
				{
					{Key: "Math", Interval: iv(14, 15)},
					{Key: "Science", Interval: iv(9, 10)},
				},
				{
					{Key: "Math", Interval: iv(9, 10)},
				},
			},
		},
		{
			name: "GISMP classic: Three subjects with overlapping slots",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(1, 3)}},
				{Key: "Science", Intervals: []Interval[int]{iv(2, 4)}},
				{Key: "History", Intervals: []Interval[int]{iv(3, 5)}},
			},
			want: [][]ScheduleEntry[string, int]{
				{
					{Key: "Math", Interval: iv(1, 3)},
					{Key: "History", Interval: iv(3, 5)},
				},
				{
					{Key: "Science", Interval: iv(2, 4)},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GroupIntervalSchedulingMaximization(tt.groups)

			if len(got) != len(tt.want) {
				t.Errorf("got %d schedules, want %d", len(got), len(tt.want))
				return
			}

			for _, expectedSched := range tt.want {
				found := false
				for _, actualSched := range got {
					if actualSched.Equals(expectedSched) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("expected schedule %v not found in results: %v", expectedSched, got)
				}
			}
		})
	}
}
