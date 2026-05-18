package pipeline_test

import "testing"

func TestAdd(t *testing.T) {
	tests := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"1+2=3", 1, 2, 3},
		{"0+0=0", 0, 0, 0},
		{"-1+1=0", -1, 1, 0},
		{"100+200=300", 100, 200, 300},
		{"-5+-3=-8", -5, -3, -8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := add(tt.a, tt.b)
			if got != tt.expected {
				t.Errorf("add(%d, %d) = %d; want %d", tt.a, tt.b, got, tt.expected)
			}
		})
	}
}
