package interval

import (
	"testing"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		start   int
		end     int
		wantErr bool
	}{
		{"valid", 1, 10, false},
		{"zero length", 5, 5, false},
		{"invalid", 10, 1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.start, tt.end)
			if (err != nil) != tt.wantErr {
				t.Errorf("New(%v, %v) error = %v, wantErr %v", tt.start, tt.end, err, tt.wantErr)
			}
		})
	}
}

func TestIntervalMethods(t *testing.T) {
	i1_5, _ := New(1, 5)
	i5_10, _ := New(5, 10)
	i2_4, _ := New(2, 4)
	i0_10, _ := New(0, 10)

	t.Run("Before", func(t *testing.T) {
		tests := []struct {
			a, b Interval[int]
			want bool
		}{
			{i1_5, i5_10, true},
			{i5_10, i1_5, false},
			{i1_5, i1_5, false},
		}
		for _, tt := range tests {
			if got := tt.a.Before(tt.b); got != tt.want {
				t.Errorf("%v.Before(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("After", func(t *testing.T) {
		tests := []struct {
			a, b Interval[int]
			want bool
		}{
			{i5_10, i1_5, true},
			{i1_5, i5_10, false},
			{i1_5, i1_5, false},
		}
		for _, tt := range tests {
			if got := tt.a.After(tt.b); got != tt.want {
				t.Errorf("%v.After(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("Touches", func(t *testing.T) {
		tests := []struct {
			a, b Interval[int]
			want bool
		}{
			{i1_5, i5_10, true},
			{i5_10, i1_5, true},
			{i1_5, i2_4, false},
			{i1_5, i0_10, false},
		}
		for _, tt := range tests {
			if got := tt.a.Touches(tt.b); got != tt.want {
				t.Errorf("%v.Touches(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})

	t.Run("Overlaps", func(t *testing.T) {
		tests := []struct {
			a, b Interval[int]
			want bool
		}{
			{i1_5, i2_4, true},
			{i1_5, i5_10, false},
			{i1_5, i0_10, true},
			{i5_10, i0_10, true},
			{i1_5, i1_5, true},
		}
		for _, tt := range tests {
			if got := tt.a.Overlaps(tt.b); got != tt.want {
				t.Errorf("%v.Overlaps(%v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		}
	})
}

func TestRelationship(t *testing.T) {
	tests := []struct {
		name string
		a, b [2]int
		want Relationship
	}{
		{"StrictlyBefore", [2]int{0, 2}, [2]int{5, 10}, StrictlyBefore},
		{"TouchingBefore", [2]int{0, 5}, [2]int{5, 10}, TouchingBefore},
		{"PartialOverlapBefore", [2]int{0, 7}, [2]int{5, 10}, PartialOverlapBefore},
		{"ContainsTouchingEnd", [2]int{0, 10}, [2]int{5, 10}, ContainsTouchingEnd},
		{"ContainsInMiddle", [2]int{0, 15}, [2]int{5, 10}, ContainsInMiddle},
		{"ContainedWithinTouchingStart", [2]int{5, 7}, [2]int{5, 10}, ContainedWithinTouchingStart},
		{"ContainsTouchingStart", [2]int{5, 15}, [2]int{5, 10}, ContainsTouchingStart},
		{"Equals", [2]int{5, 10}, [2]int{5, 10}, Equals},
		{"ContainedWithinInMiddle", [2]int{6, 9}, [2]int{5, 10}, ContainedWithinInMiddle},
		{"ContainedWithinTouchingEnd", [2]int{7, 10}, [2]int{5, 10}, ContainedWithinTouchingEnd},
		{"PartialOverlapAfter", [2]int{7, 15}, [2]int{5, 10}, PartialOverlapAfter},
		{"TouchingAfter", [2]int{10, 15}, [2]int{5, 10}, TouchingAfter},
		{"StrictlyAfter", [2]int{15, 20}, [2]int{5, 10}, StrictlyAfter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a, _ := New(tt.a[0], tt.a[1])
			b, _ := New(tt.b[0], tt.b[1])
			if got := a.Relationship(b); got != tt.want {
				t.Errorf("Relationship() = %v, want %v", got, tt.want)
			}
		})
	}
}
