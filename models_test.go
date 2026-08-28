package main

import (
	"models/pricing"
	"testing"
)

func TestDecimalPerMiTokenMarshalJSON(t *testing.T) {
	for _, test := range []struct {
		name     string
		d        pricing.Decimal
		expected string
	}{
		{
			name:     "",
			d:        pricing.NewDecimal(0, 1),
			expected: "0.1",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(0, 0, 2),
			expected: "0.02",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(0),
			expected: "0",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(2, 2, 5),
			expected: "25",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(1, 2, 3),
			expected: "2.3",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, _ := test.d.MarshalJSON()
			got := string(b)
			if got != test.expected {
				t.Fatalf("\nexp: %q\ngot: %q", test.expected, got)
			}
		})
	}
}

func TestDecimalPerTokenMarshalJSON(t *testing.T) {
	for _, test := range []struct {
		name     string
		d        pricing.Decimal
		expected string
	}{
		{
			name:     "",
			d:        pricing.NewDecimal(0, 1).PerToken(),
			expected: "0.0000001",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(0, 0, 2).PerToken(),
			expected: "0.00000002",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(0).PerToken(),
			expected: "0",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(2, 2, 5).PerToken(),
			expected: "0.000025",
		},
		{
			name:     "",
			d:        pricing.NewDecimal(1, 2, 3).PerToken(),
			expected: "0.0000023",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			b, _ := test.d.MarshalJSON()
			got := string(b)
			if got != test.expected {
				t.Fatalf("\nexp: %q\ngot: %q", test.expected, got)
			}
		})
	}
}

func TestDecimalLess(t *testing.T) {
	for _, test := range []struct {
		name     string
		a, b     pricing.Decimal
		expected bool
	}{
		{name: "", a: pricing.NewDecimal(0, 1), b: pricing.NewDecimal(0, 2), expected: true},
		{name: "", a: pricing.NewDecimal(0, 2), b: pricing.NewDecimal(0, 1), expected: false},
		{name: "", a: pricing.NewDecimal(1, 5), b: pricing.NewDecimal(2, 2, 5), expected: true},
		{name: "", a: pricing.NewDecimal(2, 2, 5), b: pricing.NewDecimal(1, 5), expected: false},
		{name: "", a: pricing.NewDecimal(1, 1, 0, 5), b: pricing.NewDecimal(1, 1, 5), expected: true},
		{name: "", a: pricing.NewDecimal(0, 0, 2), b: pricing.NewDecimal(0, 1), expected: true},
		{name: "", a: pricing.NewDecimal(0), b: pricing.NewDecimal(0, 1), expected: true},
		{name: "", a: pricing.NewDecimal(0), b: pricing.NewDecimal(0), expected: false},
		{name: "", a: pricing.NewDecimal(1, 2, 3), b: pricing.NewDecimal(1, 2, 3), expected: false},
		{name: "", a: pricing.NewDecimal(2, 2, 5), b: pricing.NewDecimal(2, 2, 5), expected: false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.a.Less(test.b); got != test.expected {
				t.Fatalf("\nexp: %v\ngot: %v", test.expected, got)
			}
		})
	}
}
