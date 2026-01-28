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
		want   []Schedule[string, int]
	}{
		{
			name: "No conflicts: Math and Science at different times",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(9, 10)}},
				{Key: "Science", Intervals: []Interval[int]{iv(11, 12)}},
			},
			want: []Schedule[string, int]{
				{
					"Math":    iv(9, 10),
					"Science": iv(11, 12),
				},
			},
		},
		{
			name: "Simple conflict: Math and History overlap",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(9, 11)}},
				{Key: "History", Intervals: []Interval[int]{iv(10, 12)}},
			},
			want: []Schedule[string, int]{
				{"Math": iv(9, 11)},
				{"History": iv(10, 12)},
			},
		},
		{
			name: "Multiple options: Math has two slots, one overlaps with Science",
			groups: []Group[string, int]{
				{
					Key:       "Math",
					Intervals: []Interval[int]{iv(9, 10), iv(14, 15)},
				},
				{
					Key:       "Science",
					Intervals: []Interval[int]{iv(9, 10)},
				},
			},
			want: []Schedule[string, int]{
				{
					"Math":    iv(14, 15),
					"Science": iv(9, 10),
				},
				{
					"Math": iv(9, 10),
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
			want: []Schedule[string, int]{
				{
					"Math":    iv(1, 3),
					"History": iv(3, 5),
				},
				{
					"Science": iv(2, 4),
				},
			},
		},
		{
			name: "Nested intervals (Containment)",
			groups: []Group[string, int]{
				{Key: "Lecture", Intervals: []Interval[int]{iv(0, 10)}},
				{Key: "Lab", Intervals: []Interval[int]{iv(1, 2)}},
				{Key: "Seminar", Intervals: []Interval[int]{iv(3, 4)}},
				{Key: "Workshop", Intervals: []Interval[int]{iv(5, 6)}},
				{Key: "Tutorial", Intervals: []Interval[int]{iv(7, 8)}},
			},
			want: []Schedule[string, int]{
				{
					"Lecture": iv(0, 10),
				},
				{
					"Lab":      iv(1, 2),
					"Seminar":  iv(3, 4),
					"Workshop": iv(5, 6),
					"Tutorial": iv(7, 8),
				},
			},
		},
		{
			name: "5 groups with complex overlaps",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(8, 10)}},
				{Key: "Science", Intervals: []Interval[int]{iv(9, 11)}},
				{Key: "History", Intervals: []Interval[int]{iv(10, 12)}},
				{Key: "Art", Intervals: []Interval[int]{iv(8, 12)}},
				{Key: "Music", Intervals: []Interval[int]{iv(13, 14)}},
			},
			want: []Schedule[string, int]{
				{
					"Math":    iv(8, 10),
					"History": iv(10, 12),
					"Music":   iv(13, 14),
				},
				{
					"Science": iv(9, 11),
					"Music":   iv(13, 14),
				},
				{
					"Art":   iv(8, 12),
					"Music": iv(13, 14),
				},
			},
		},
		{
			name: "7 groups, 3 intervals each: Perfect packing vs. Long overlaps",
			groups: []Group[string, int]{
				{Key: "Math", Intervals: []Interval[int]{iv(0, 1), iv(0, 10), iv(0, 11)}},
				{Key: "Science", Intervals: []Interval[int]{iv(1, 2), iv(0, 10), iv(0, 11)}},
				{Key: "History", Intervals: []Interval[int]{iv(2, 3), iv(0, 10), iv(0, 11)}},
				{Key: "Art", Intervals: []Interval[int]{iv(3, 4), iv(0, 10), iv(0, 11)}},
				{Key: "Music", Intervals: []Interval[int]{iv(4, 5), iv(0, 10), iv(0, 11)}},
				{Key: "PE", Intervals: []Interval[int]{iv(5, 6), iv(0, 10), iv(0, 11)}},
				{Key: "English", Intervals: []Interval[int]{iv(6, 7), iv(0, 10), iv(0, 11)}},
			},
			want: []Schedule[string, int]{
				{
					"Math":    iv(0, 1),
					"Science": iv(1, 2),
					"History": iv(2, 3),
					"Art":     iv(3, 4),
					"Music":   iv(4, 5),
					"PE":      iv(5, 6),
					"English": iv(6, 7),
				},
				{"Math": iv(0, 10)},
				{"Math": iv(0, 11)},
				{"Science": iv(0, 10)},
				{"Science": iv(0, 11)},
				{"History": iv(0, 10)},
				{"History": iv(0, 11)},
				{"Art": iv(0, 10)},
				{"Art": iv(0, 11)},
				{"Music": iv(0, 10)},
				{"Music": iv(0, 11)},
				{"PE": iv(0, 10)},
				{"PE": iv(0, 11)},
				{"English": iv(0, 10)},
				{"English": iv(0, 11)},
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
			name: "same entries",
			s1: Schedule[string, int]{
				"Math":    iv(9, 10),
				"Science": iv(11, 12),
			},
			s2: Schedule[string, int]{
				"Math":    iv(9, 10),
				"Science": iv(11, 12),
			},
			want: true,
		},
		{
			name: "different lengths",
			s1: Schedule[string, int]{
				"Math": iv(9, 10),
			},
			s2: Schedule[string, int]{
				"Math":    iv(9, 10),
				"Science": iv(11, 12),
			},
			want: false,
		},
		{
			name: "same length different keys",
			s1: Schedule[string, int]{
				"Math": iv(9, 10),
			},
			s2: Schedule[string, int]{
				"Science": iv(9, 10),
			},
			want: false,
		},
		{
			name: "same length different intervals",
			s1: Schedule[string, int]{
				"Math": iv(9, 10),
			},
			s2: Schedule[string, int]{
				"Math": iv(10, 11),
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

func TestSchedule_Overlaps(t *testing.T) {
	iv := func(start, end int) Interval[int] {
		i, err := New(start, end)
		if err != nil {
			t.Fatalf("New(%v, %v) failed: %v", start, end, err)
		}
		return i
	}

	tests := []struct {
		name     string
		schedule Schedule[string, int]
		interval Interval[int]
		want     bool
	}{
		{
			name:     "empty schedule",
			schedule: nil,
			interval: iv(1, 2),
			want:     false,
		},
		{
			name: "non-overlapping interval",
			schedule: Schedule[string, int]{
				"Math": iv(1, 2),
				"Art":  iv(4, 5),
			},
			interval: iv(2, 4),
			want:     false,
		},
		{
			name: "touching interval",
			schedule: Schedule[string, int]{
				"Math": iv(1, 2),
			},
			interval: iv(2, 3),
			want:     false,
		},
		{
			name: "overlapping interval",
			schedule: Schedule[string, int]{
				"Math": iv(1, 3),
			},
			interval: iv(2, 4),
			want:     true,
		},
		{
			name: "contained interval",
			schedule: Schedule[string, int]{
				"Math": iv(1, 5),
			},
			interval: iv(2, 4),
			want:     true,
		},
		{
			name: "overlaps one of many",
			schedule: Schedule[string, int]{
				"Math":    iv(1, 2),
				"Science": iv(3, 6),
				"Art":     iv(7, 8),
			},
			interval: iv(5, 7),
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.schedule.Overlaps(tt.interval); got != tt.want {
				t.Errorf("schedule.Overlaps(%v) = %v, want %v", tt.interval, got, tt.want)
			}
		})
	}
}

func TestSchedule_OrderedKeys(t *testing.T) {
	iv := func(start, end int) Interval[int] {
		i, err := New(start, end)
		if err != nil {
			t.Fatalf("New(%v, %v) failed: %v", start, end, err)
		}
		return i
	}

	tests := []struct {
		name string
		s    Schedule[string, int]
		want []string
	}{
		{
			name: "empty schedule",
			s:    Schedule[string, int]{},
			want: []string{},
		},
		{
			name: "single entry",
			s: Schedule[string, int]{
				"Math": iv(9, 10),
			},
			want: []string{"Math"},
		},
		{
			name: "some schedule",
			s: Schedule[string, int]{
				"History": iv(11, 12),
				"Math":    iv(9, 10),
				"Science": iv(10, 11),
			},
			want: []string{"Math", "Science", "History"},
		},
		{
			name: "negative",
			s: Schedule[string, int]{
				"Afternoon": iv(13, 14),
				"Early":     iv(-1, 1),
				"Morning":   iv(8, 9),
			},
			want: []string{"Early", "Morning", "Afternoon"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.s.OrderedKeys()
			if len(got) != len(tt.want) {
				t.Fatalf("got %d keys, want %d", len(got), len(tt.want))
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("key %d = %q, want %q (got %v)", i, got[i], tt.want[i], got)
				}
			}
		})
	}
}
