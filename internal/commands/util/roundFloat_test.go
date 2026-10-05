package util

import (
	"math"
	"testing"
)

func TestRoundFloat_BasicRounding(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Round to 2 decimal places",
			value:     3.14159,
			precision: 2,
			expected:  3.14,
		},
		{
			name:      "Round to 1 decimal place",
			value:     2.567,
			precision: 1,
			expected:  2.6,
		},
		{
			name:      "Round to 3 decimal places",
			value:     1.23456,
			precision: 3,
			expected:  1.235,
		},
		{
			name:      "Round to 0 decimal places",
			value:     5.7,
			precision: 0,
			expected:  6,
		},
		{
			name:      "Round to 0 decimal places (down)",
			value:     5.4,
			precision: 0,
			expected:  5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if got != tt.expected {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_NegativeNumbers(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Negative number to 2 decimal places",
			value:     -3.14159,
			precision: 2,
			expected:  -3.14,
		},
		{
			name:      "Negative number to 1 decimal place",
			value:     -2.567,
			precision: 1,
			expected:  -2.6,
		},
		{
			name:      "Negative number to 0 decimal places",
			value:     -5.7,
			precision: 0,
			expected:  -6,
		},
		{
			name:      "Negative number to 0 decimal places (down)",
			value:     -5.4,
			precision: 0,
			expected:  -5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if got != tt.expected {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_ZeroValue(t *testing.T) {
	tests := []struct {
		name      string
		precision uint
	}{
		{
			name:      "Zero with 0 precision",
			precision: 0,
		},
		{
			name:      "Zero with 2 precision",
			precision: 2,
		},
		{
			name:      "Zero with 5 precision",
			precision: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(0.0, tt.precision)
			if got != 0.0 {
				t.Errorf("RoundFloat(0.0, %d) = %v, want 0.0", tt.precision, got)
			}
		})
	}
}

func TestRoundFloat_HighPrecision(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Round to 5 decimal places",
			value:     1.234567,
			precision: 5,
			expected:  1.23457,
		},
		{
			name:      "Round to 10 decimal places",
			value:     3.141592653589793,
			precision: 10,
			expected:  3.1415926536,
		},
		{
			name:      "Round pi to 7 decimal places",
			value:     math.Pi,
			precision: 7,
			expected:  3.1415927,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if !almostEqual(got, tt.expected, 1e-10) {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_LargeNumbers(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Large number to 2 decimal places",
			value:     123456.789,
			precision: 2,
			expected:  123456.79,
		},
		{
			name:      "Large number to 0 decimal places",
			value:     999999.5,
			precision: 0,
			expected:  1000000,
		},
		{
			name:      "Large number with many decimals",
			value:     1234567.123456,
			precision: 3,
			expected:  1234567.123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if !almostEqual(got, tt.expected, 1e-6) {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_SmallNumbers(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Small number to 5 decimal places",
			value:     0.00012345,
			precision: 5,
			expected:  0.00012,
		},
		{
			name:      "Very small number to 10 decimal places",
			value:     1e-8,
			precision: 10,
			expected:  1e-8,
		},
		{
			name:      "Small number to 3 decimal places",
			value:     0.0009,
			precision: 3,
			expected:  0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if !almostEqual(got, tt.expected, 1e-12) {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_Idempotent(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
	}{
		{
			name:      "Already rounded value at 2 precision",
			value:     3.14,
			precision: 2,
		},
		{
			name:      "Already rounded value at 0 precision",
			value:     5.0,
			precision: 0,
		},
		{
			name:      "Already rounded value at 4 precision",
			value:     1.2345,
			precision: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rounded1 := RoundFloat(tt.value, tt.precision)
			rounded2 := RoundFloat(rounded1, tt.precision)
			if rounded1 != rounded2 {
				t.Errorf("RoundFloat is not idempotent: first=%v, second=%v", rounded1, rounded2)
			}
		})
	}
}

func TestRoundFloat_NearBoundary(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
		expected  float64
	}{
		{
			name:      "Round 0.5 to 0 decimal places",
			value:     0.5,
			precision: 0,
			expected:  1,
		},
		{
			name:      "Round 1.5 to 0 decimal places",
			value:     1.5,
			precision: 0,
			expected:  2,
		},
		{
			name:      "Round 2.5 to 0 decimal places",
			value:     2.5,
			precision: 0,
			expected:  3,
		},
		{
			name:      "Round 3.5 to 0 decimal places",
			value:     3.5,
			precision: 0,
			expected:  4,
		},
		{
			name:      "Round 0.005 to 2 decimal places",
			value:     0.005,
			precision: 2,
			expected:  0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if !almostEqual(got, tt.expected, 1e-10) {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", tt.value, tt.precision, got, tt.expected)
			}
		})
	}
}

func TestRoundFloat_SpecialValues(t *testing.T) {
	tests := []struct {
		name      string
		value     float64
		precision uint
	}{
		{
			name:      "Positive infinity",
			value:     math.Inf(1),
			precision: 2,
		},
		{
			name:      "Negative infinity",
			value:     math.Inf(-1),
			precision: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RoundFloat(tt.value, tt.precision)
			if !math.IsInf(got, 0) {
				t.Errorf("RoundFloat(%v, %d) = %v, want infinity", tt.value, tt.precision, got)
			}
		})
	}
}

func TestRoundFloat_VaryingPrecisions(t *testing.T) {
	value := 12.3456789
	tests := []struct {
		precision uint
		expected  float64
	}{
		{0, 12},
		{1, 12.3},
		{2, 12.35},
		{3, 12.346},
		{4, 12.3457},
		{5, 12.34568},
		{6, 12.345679},
	}

	for _, tt := range tests {
		t.Run("Precision"+string(rune(tt.precision+'0')), func(t *testing.T) {
			got := RoundFloat(value, tt.precision)
			if !almostEqual(got, tt.expected, 1e-8) {
				t.Errorf("RoundFloat(%v, %d) = %v, want %v", value, tt.precision, got, tt.expected)
			}
		})
	}
}

// Helper function to compare floats with a tolerance for floating-point precision errors
func almostEqual(a, b, tolerance float64) bool {
	if math.IsInf(a, 0) || math.IsInf(b, 0) {
		return (math.IsInf(a, 1) && math.IsInf(b, 1)) || (math.IsInf(a, -1) && math.IsInf(b, -1))
	}
	diff := math.Abs(a - b)
	return diff < tolerance
}
