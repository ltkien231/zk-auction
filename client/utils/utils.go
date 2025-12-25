package utils

import (
	"crypto/rand"
	"math/big"
)

func RandBigInt(max *big.Int) *big.Int {
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		panic("Failed to generate random big.Int") // TODO: handle error properly
	}
	return n
}

func BitsToInt(bits []int) int {
	var result int
	for _, bit := range bits {
		result <<= 1
		if bit == 1 {
			result |= 1
		}
	}
	return result
}

func IntToBits(value int, bitLength int) []int {
	bits := make([]int, bitLength)
	for i := bitLength - 1; i >= 0; i-- {
		bits[i] = value & 1
		value >>= 1
	}
	return bits
}
