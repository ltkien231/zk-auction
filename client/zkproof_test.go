package client

import (
	"fmt"
	"math/big"
	"testing"
)

// TestZKProofEij_Case0 tests the ZK proof when b_ij = 0
// In this case, e_ij = g^{t_ij} * h^{s_ij}
func TestZKProofEij_Case0(t *testing.T) {
	// Setup system parameters (using small primes for testing)
	params := setupTestParams()

	// Bidder's secret values
	b_i := big.NewInt(5)  // Bidder's bid (for commitment)
	r_i := big.NewInt(7)  // Randomness for commitment
	t_ij := big.NewInt(3) // Secret value from Round I
	s_ij := big.NewInt(4) // Secret value from Round I
	b_ij := 0             // j-th bit of bid is 0
	j := 2                // Bit position

	// Create bidder's commitment C_i = g^{b_i} * h^{r_i}
	C_i := &BidCommitment{
		C: computeCommitment(params, b_i, r_i),
	}

	// Compute e_ij = g^{t_ij} * h^{s_ij} (since b_ij = 0)
	e_ij := new(big.Int).Exp(params.G, t_ij, params.P)
	temp := new(big.Int).Exp(params.H, s_ij, params.P)
	e_ij.Mul(e_ij, temp)
	e_ij.Mod(e_ij, params.P)

	// Generate ZK proof
	proof, err := GenerateZKProofEij(params, C_i, e_ij, t_ij, s_ij, b_ij, j)
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify the proof
	isValid := VerifyZKProofEij(params, C_i, e_ij, proof, j)
	if !isValid {
		t.Errorf("Proof verification failed for b_ij = 0")
	} else {
		fmt.Println("✓ ZK Proof for b_ij = 0 is valid!")
	}

	// Test that proof fails with wrong bit position
	isValid = VerifyZKProofEij(params, C_i, e_ij, proof, j+1)
	if isValid {
		t.Errorf("Proof should fail with different bit position")
	} else {
		fmt.Println("✓ Proof correctly fails with wrong bit position")
	}
}

// TestZKProofEij_Case1 tests the ZK proof when b_ij = 1
// In this case, e_ij = g^{t_ij} * h^{s_ij} * g
func TestZKProofEij_Case1(t *testing.T) {
	// Setup system parameters
	params := setupTestParams()

	// Bidder's secret values
	b_i := big.NewInt(13) // Bidder's bid (for commitment)
	r_i := big.NewInt(11) // Randomness for commitment
	t_ij := big.NewInt(8) // Secret value from Round I
	s_ij := big.NewInt(6) // Secret value from Round I
	b_ij := 1             // j-th bit of bid is 1
	j := 3                // Bit position

	// Create bidder's commitment C_i = g^{b_i} * h^{r_i}
	C_i := &BidCommitment{
		C: computeCommitment(params, b_i, r_i),
	}

	// Compute e_ij = g^{t_ij} * h^{s_ij} * g (since b_ij = 1)
	e_ij := new(big.Int).Exp(params.G, t_ij, params.P)
	temp := new(big.Int).Exp(params.H, s_ij, params.P)
	e_ij.Mul(e_ij, temp)
	e_ij.Mul(e_ij, params.G) // Multiply by g
	e_ij.Mod(e_ij, params.P)

	// Generate ZK proof
	proof, err := GenerateZKProofEij(params, C_i, e_ij, t_ij, s_ij, b_ij, j)
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Verify the proof
	isValid := VerifyZKProofEij(params, C_i, e_ij, proof, j)
	if !isValid {
		t.Errorf("Proof verification failed for b_ij = 1")
	} else {
		fmt.Println("✓ ZK Proof for b_ij = 1 is valid!")
	}
}

// TestZKProofEij_InvalidBit tests error handling for invalid bit values
func TestZKProofEij_InvalidBit(t *testing.T) {
	params := setupTestParams()

	C_i := &BidCommitment{C: big.NewInt(100)}
	e_ij := big.NewInt(200)
	t_ij := big.NewInt(3)
	s_ij := big.NewInt(4)
	b_ij := 2 // Invalid bit value (should be 0 or 1)
	j := 1

	// Should return error for invalid bit value
	_, err := GenerateZKProofEij(params, C_i, e_ij, t_ij, s_ij, b_ij, j)
	if err == nil {
		t.Errorf("Expected error for invalid bit value, got nil")
	} else {
		fmt.Printf("✓ Correctly caught invalid bit value: %v\n", err)
	}
}

// TestZKProofEij_WrongEij tests that proof fails with incorrectly constructed e_ij
func TestZKProofEij_WrongEij(t *testing.T) {
	params := setupTestParams()

	// Setup values
	b_i := big.NewInt(5)
	r_i := big.NewInt(7)
	t_ij := big.NewInt(3)
	s_ij := big.NewInt(4)
	b_ij := 0
	j := 2

	C_i := &BidCommitment{
		C: computeCommitment(params, b_i, r_i),
	}

	// Create CORRECT e_ij for b_ij = 0
	e_ij_correct := new(big.Int).Exp(params.G, t_ij, params.P)
	temp := new(big.Int).Exp(params.H, s_ij, params.P)
	e_ij_correct.Mul(e_ij_correct, temp)
	e_ij_correct.Mod(e_ij_correct, params.P)

	// Create WRONG e_ij (pretend it's for b_ij = 1 instead)
	e_ij_wrong := new(big.Int).Set(e_ij_correct)
	e_ij_wrong.Mul(e_ij_wrong, params.G)
	e_ij_wrong.Mod(e_ij_wrong, params.P)

	// Generate proof for correct e_ij
	proof, err := GenerateZKProofEij(params, C_i, e_ij_correct, t_ij, s_ij, b_ij, j)
	if err != nil {
		t.Fatalf("Failed to generate proof: %v", err)
	}

	// Try to verify with WRONG e_ij - should fail
	isValid := VerifyZKProofEij(params, C_i, e_ij_wrong, proof, j)
	if isValid {
		t.Errorf("Proof should fail with wrong e_ij")
	} else {
		fmt.Println("✓ Proof correctly fails with wrong e_ij")
	}
}

// ExampleZKProofEij demonstrates the complete workflow
func ExampleZKProofEij() {
	fmt.Print("=== ZK Proof for e_ij Example ===\n\n")

	// 1. Setup system parameters
	params := setupTestParams()
	fmt.Printf("System parameters:\n")
	fmt.Printf("  p = %s (prime)\n", params.P.String())
	fmt.Printf("  q = %s (prime order)\n", params.Q.String())
	fmt.Printf("  g = %s (generator)\n", params.G.String())
	fmt.Printf("  h = %s (generator)\n\n", params.H.String())

	// 2. Bidder prepares his data
	b_ij := 0             // The j-th bit is 0
	j := 1                // Bit position
	t_ij := big.NewInt(3) // Random secret
	s_ij := big.NewInt(4) // Random secret

	fmt.Printf("Bidder's secret data:\n")
	fmt.Printf("  b_ij = %d (bit value)\n", b_ij)
	fmt.Printf("  j = %d (bit position)\n", j)
	fmt.Printf("  t_ij = %s\n", t_ij.String())
	fmt.Printf("  s_ij = %s\n\n", s_ij.String())

	// 3. Compute e_ij based on b_ij
	e_ij := new(big.Int).Exp(params.G, t_ij, params.P)
	temp := new(big.Int).Exp(params.H, s_ij, params.P)
	e_ij.Mul(e_ij, temp)
	if b_ij == 1 {
		e_ij.Mul(e_ij, params.G)
	}
	e_ij.Mod(e_ij, params.P)

	fmt.Printf("Computed e_ij = %s\n\n", e_ij.String())

	// 4. Create commitment (for demonstration)
	C_i := &BidCommitment{C: big.NewInt(12345)}

	// 5. Generate ZK proof
	proof, err := GenerateZKProofEij(params, C_i, e_ij, t_ij, s_ij, b_ij, j)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Generated ZK Proof:\n")
	fmt.Printf("  c1 = %s\n", proof.C1.String())
	fmt.Printf("  c2 = %s\n", proof.C2.String())
	fmt.Printf("  z1 = %s\n", proof.Z1.String())
	fmt.Printf("  z2 = %s\n", proof.Z2.String())
	fmt.Printf("  a1 = %s\n", proof.A1.String())
	fmt.Printf("  a2 = %s\n\n", proof.A2.String())

	// 6. Verify the proof
	isValid := VerifyZKProofEij(params, C_i, e_ij, proof, j)
	fmt.Printf("Proof verification: %v\n", isValid)

	if isValid {
		fmt.Println("\n✓ The verifier is convinced that e_ij is correctly constructed")
		fmt.Println("✓ But the verifier does NOT know whether b_ij = 0 or b_ij = 1!")
		fmt.Print("✓ This is the zero-knowledge property!")
	}
}

// setupTestParams creates test system parameters
// WARNING: These are SMALL values for testing only!
// In production, use large cryptographically secure primes
func setupTestParams() *SystemParams {
	// Using small primes for testing (DO NOT use in production!)
	// p = 23, q = 11 (where q | (p-1))
	p := big.NewInt(23)
	q := big.NewInt(11)
	g := big.NewInt(5) // Generator of order q
	h := big.NewInt(7) // Another generator (should be g^x for unknown x)

	return &SystemParams{
		P: p,
		Q: q,
		G: g,
		H: h,
	}
}

// computeCommitment computes Pederson commitment C = g^m * h^r
func computeCommitment(params *SystemParams, message, randomness *big.Int) *big.Int {
	// C = g^m * h^r mod p
	commitment := new(big.Int).Exp(params.G, message, params.P)
	temp := new(big.Int).Exp(params.H, randomness, params.P)
	commitment.Mul(commitment, temp)
	commitment.Mod(commitment, params.P)
	return commitment
}
