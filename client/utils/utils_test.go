package utils

import (
	"math/big"
	"testing"
)

func TestRanBigInt(t *testing.T) {
	max := big.NewInt(100)
	n := RandBigInt(max)
	if n.Cmp(big.NewInt(0)) < 0 || n.Cmp(max) >= 0 {
		t.Errorf("Generated number %s out of range [0, %s)", n.String(), max.String())
	}
}

func TestBitsToInt(t *testing.T) {
	testCases := []struct {
		binary   []int
		expected int
	}{
		{binary: []int{1, 0, 1, 1}, expected: 11},
		{binary: []int{0, 0, 1, 0}, expected: 2},
		{binary: []int{1, 1, 1, 1}, expected: 15},
		{binary: []int{1, 0, 0, 0}, expected: 8},
	}

	for _, tc := range testCases {
		result := BitsToInt(tc.binary)
		if result != tc.expected {
			t.Errorf("BitsToInt(%v) = %d; want %d", tc.binary, result, tc.expected)
		}
	}
}

func TestIntToBits(t *testing.T) {
	testCases := []struct {
		value     int
		bitLength int
		expected  []int
	}{
		{value: 11, bitLength: 4, expected: []int{1, 0, 1, 1}},
		{value: 2, bitLength: 4, expected: []int{0, 0, 1, 0}},
		{value: 15, bitLength: 4, expected: []int{1, 1, 1, 1}},
		{value: 8, bitLength: 4, expected: []int{1, 0, 0, 0}},
	}

	for _, tc := range testCases {
		result := IntToBits(tc.value, tc.bitLength)
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("IntToBits(%d, %d) = %v; want %v", tc.value, tc.bitLength, result, tc.expected)
				break
			}
		}
	}
}
