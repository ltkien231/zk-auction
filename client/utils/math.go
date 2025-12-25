package utils

import "math/big"

// Modular addition
func AddMod(a, b, mod *big.Int) *big.Int {
	result := new(big.Int).Add(a, b)
	return result.Mod(result, mod)
}

// Modular subtraction
func SubMod(a, b, mod *big.Int) *big.Int {
	result := new(big.Int).Sub(a, b)
	result.Mod(result, mod)
	// Handle negative results
	if result.Sign() < 0 {
		result.Add(result, mod)
	}
	return result
}

// Modular multiplication
func MulMod(a, b, mod *big.Int) *big.Int {
	result := new(big.Int).Mul(a, b)
	return result.Mod(result, mod)
}

// Modular division
func DivMod(a, b, mod *big.Int) *big.Int {
	bInv := new(big.Int).ModInverse(b, mod)
	if bInv == nil {
		panic("Division by zero in modular arithmetic")
	}
	return MulMod(a, bInv, mod)
}
