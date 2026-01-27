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

func TestSchedule_Equals(t *testing.T) {
	iv := func(start, end int) Interval[int] {
		i, err := New(start, end)
		if err != nil {
			t.Fatalf("New(%v, %v) failed: %v", start, end, err)
		}
		return i
	}

	tests := []struct {
		name string
		s1   Schedule[string, int]
		s2   Schedule[string, int]
		want bool
	}{
		{
			name: "empty schedules",
			s1:   Schedule[string, int]{},
			s2:   Schedule[string, int]{},
			want: true,
		},
		{
			name: "same entries same order",
			s1: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
				{Key: "Science", Interval: iv(11, 12)},
			},
			s2: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
				{Key: "Science", Interval: iv(11, 12)},
			},
			want: true,
		},
		{
			name: "same entries different order",
			s1: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
				{Key: "Science", Interval: iv(11, 12)},
			},
			s2: Schedule[string, int]{
				{Key: "Science", Interval: iv(11, 12)},
				{Key: "Math", Interval: iv(9, 10)},
			},
			want: true,
		},
		{
			name: "different lengths",
			s1: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
			},
			s2: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
				{Key: "Science", Interval: iv(11, 12)},
			},
			want: false,
		},
		{
			name: "same length different keys",
			s1: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
			},
			s2: Schedule[string, int]{
				{Key: "Science", Interval: iv(9, 10)},
			},
			want: false,
		},
		{
			name: "same length different intervals",
			s1: Schedule[string, int]{
				{Key: "Math", Interval: iv(9, 10)},
			},
			s2: Schedule[string, int]{
				{Key: "Math", Interval: iv(10, 11)},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.s1.Equals(tt.s2); got != tt.want {
				t.Errorf("s1.Equals(s2) = %v, want %v", got, tt.want)
			}
			if got := tt.s2.Equals(tt.s1); got != tt.want {
				t.Errorf("s2.Equals(s1) = %v, want %v (symmetry check)", got, tt.want)
			}
		})
	}
}
