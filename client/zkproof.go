package client

import (
	"crypto/rand"
	"crypto/sha256"
	"math/big"
)

// ZKProofEij represents the Non-Interactive Zero-Knowledge Proof for e_ij
// This proves that e_ij is correctly constructed as either:
// - g^{t_ij} * h^{s_ij} (when b_ij = 0)
// - g^{t_ij} * h^{s_ij} * g (when b_ij = 1)
// without revealing which case it is or the secret values t_ij, s_ij
//
// The proof uses OR-composition: one branch is real, one is simulated.
// For the real branch, we use (z1, z2) as responses.
// For the fake branch, we use (w, v) as responses.
type ZKProofEij struct {
	C1 *big.Int // Challenge for the first equation
	C2 *big.Int // Challenge for the second equation
	Z1 *big.Int // Response z1 for real case
	Z2 *big.Int // Response z2 for real case
	W  *big.Int // Response w for fake case
	V  *big.Int // Response v for fake case
	A1 *big.Int // Commitment for equation 1
	A2 *big.Int // Commitment for equation 2
}

// SystemParams contains the public parameters of the auction system
type SystemParams struct {
	G *big.Int // Generator g of the cyclic group
	H *big.Int // Generator h = g^x where x is unknown
	Q *big.Int // Prime order of the group
	P *big.Int // Prime p where q | (p-1)
}

// BidCommitment represents the Pederson commitment of bidder's bid
type BidCommitment struct {
	C *big.Int // Commitment C_i = g^{b_i} * h^{r_i}
}

// GenerateZKProofEij generates a NIZK proof for e_ij
// Parameters:
//   - params: System parameters (g, h, q, p)
//   - C_i: Bidder's commitment
//   - e_ij: The value to prove (either g^{t_ij}*h^{s_ij} or g^{t_ij}*h^{s_ij}*g)
//   - t_ij: Secret value from Round I (randomly chosen)
//   - s_ij: Secret value from Round I (randomly chosen)
//   - b_ij: The j-th bit of bidder i's bid (0 or 1)
//   - j: Bit position (used in hash to prevent replay attacks)
//
// Returns: ZKProofEij or error
func GenerateZKProofEij(params *SystemParams, C_i *BidCommitment, e_ij *big.Int, t_ij, s_ij *big.Int, b_ij int, j int) (*ZKProofEij, error) {
	// Generate random values α, β, w, v ∈ Z_q
	alpha, err := randBigInt(params.Q)
	if err != nil {
		return nil, err
	}
	beta, err := randBigInt(params.Q)
	if err != nil {
		return nil, err
	}
	w, err := randBigInt(params.Q)
	if err != nil {
		return nil, err
	}
	v, err := randBigInt(params.Q)
	if err != nil {
		return nil, err
	}

	var a1, a2 *big.Int
	var c1, c2, z1, z2 *big.Int

	if b_ij == 0 {
		// Case 1: b_ij = 0, so e_ij = g^{t_ij} * h^{s_ij}
		// We create a REAL proof for the first case and FAKE proof for the second
		//
		// For OR-proof, we:
		// 1. Create REAL commitment a1 honestly
		// 2. Choose FAKE challenge c2 and responses w, v
		// 3. Compute FAKE commitment a2 to make equation 2 hold
		// 4. Get full challenge c from hash
		// 5. Compute REAL challenge c1 = c - c2
		// 6. Compute REAL responses z1, z2

		// Step 1: Create real commitment a1 = g^α * h^β
		a1 = new(big.Int).Exp(params.G, alpha, params.P)
		temp := new(big.Int).Exp(params.H, beta, params.P)
		a1.Mul(a1, temp)
		a1.Mod(a1, params.P)

		// Step 2: Choose fake challenge c2 and fake responses w, v
		c2, err = randBigInt(params.Q)
		if err != nil {
			return nil, err
		}

		// Step 3: Compute fake commitment a2 to satisfy equation 2
		// Equation 2: g^w * h^v = a2 * (e_ij/g)^{c2}
		// So: a2 = (g^w * h^v) / (e_ij/g)^{c2}

		// Compute g^w * h^v
		gwv := new(big.Int).Exp(params.G, w, params.P)
		temp = new(big.Int).Exp(params.H, v, params.P)
		gwv.Mul(gwv, temp)
		gwv.Mod(gwv, params.P)

		// Compute (e_ij / g)^{c2}
		gInv := new(big.Int).ModInverse(params.G, params.P)
		eijDivG := new(big.Int).Mul(e_ij, gInv)
		eijDivG.Mod(eijDivG, params.P)
		eijDivGc2 := new(big.Int).Exp(eijDivG, c2, params.P)

		// a2 = gwv / eijDivGc2
		eijDivGc2Inv := new(big.Int).ModInverse(eijDivGc2, params.P)
		a2 = new(big.Int).Mul(gwv, eijDivGc2Inv)
		a2.Mod(a2, params.P)

		// Step 4: Compute challenge c = H(g, h, C_i, e_ij, a1, a2, j)
		c := computeChallenge(params, C_i.C, e_ij, a1, a2, j)

		// Step 5: Compute real challenge c1 = c - c2
		c1 = new(big.Int).Sub(c, c2)
		c1.Mod(c1, params.Q)

		// Step 6: Compute real responses for equation 1
		// z1 = α + c1 * t_ij mod q
		z1 = new(big.Int).Mul(c1, t_ij)
		z1.Add(z1, alpha)
		z1.Mod(z1, params.Q)

		// z2 = β + c1 * s_ij mod q
		z2 = new(big.Int).Mul(c1, s_ij)
		z2.Add(z2, beta)
		z2.Mod(z2, params.Q)

		// Step 7: Fake responses for equation 2 are just w, v
		// (no computation needed, we already have them)

	} else if b_ij == 1 {
		// Case 2: b_ij = 1, so e_ij = g^{t_ij} * h^{s_ij} * g
		// We create a FAKE proof for the first case and REAL proof for the second
		//
		// For OR-proof, we:
		// 1. Choose FAKE challenge c1 and responses w, v
		// 2. Compute FAKE commitment a1 to make equation 1 hold
		// 3. Create REAL commitment a2 honestly
		// 4. Get full challenge c from hash
		// 5. Compute REAL challenge c2 = c - c1
		// 6. Compute REAL responses z1, z2

		// Step 1: Choose fake challenge c1 and fake responses w, v
		c1, err = randBigInt(params.Q)
		if err != nil {
			return nil, err
		}

		// Step 2: Compute fake commitment a1 to satisfy equation 1
		// Equation 1: g^w * h^v = a1 * e_ij^{c1}
		// So: a1 = (g^w * h^v) / e_ij^{c1}

		// Compute g^w * h^v
		gwv := new(big.Int).Exp(params.G, w, params.P)
		temp := new(big.Int).Exp(params.H, v, params.P)
		gwv.Mul(gwv, temp)
		gwv.Mod(gwv, params.P)

		// Compute e_ij^{c1}
		eijc1 := new(big.Int).Exp(e_ij, c1, params.P)

		// a1 = gwv / eijc1
		eijc1Inv := new(big.Int).ModInverse(eijc1, params.P)
		a1 = new(big.Int).Mul(gwv, eijc1Inv)
		a1.Mod(a1, params.P)

		// Step 3: Create real commitment a2 = g^α * h^β
		a2 = new(big.Int).Exp(params.G, alpha, params.P)
		temp = new(big.Int).Exp(params.H, beta, params.P)
		a2.Mul(a2, temp)
		a2.Mod(a2, params.P)

		// Step 4: Compute challenge c = H(g, h, C_i, e_ij, a1, a2, j)
		c := computeChallenge(params, C_i.C, e_ij, a1, a2, j)

		// Step 5: Compute real challenge c2 = c - c1
		c2 = new(big.Int).Sub(c, c1)
		c2.Mod(c2, params.Q)

		// Step 6: Compute real responses for equation 2
		// z1 = α + c2 * t_ij mod q
		z1 = new(big.Int).Mul(c2, t_ij)
		z1.Add(z1, alpha)
		z1.Mod(z1, params.Q)

		// z2 = β + c2 * s_ij mod q
		z2 = new(big.Int).Mul(c2, s_ij)
		z2.Add(z2, beta)
		z2.Mod(z2, params.Q)

		// Step 7: Fake responses for equation 1 are just w, v
		// (no computation needed, we already have them)

	} else {
		return nil, ErrInvalidBitValue
	}

	return &ZKProofEij{
		C1: c1,
		C2: c2,
		Z1: z1,
		Z2: z2,
		W:  w,
		V:  v,
		A1: a1,
		A2: a2,
	}, nil
}

// VerifyZKProofEij verifies the NIZK proof for e_ij
// This verification ensures that e_ij is correctly constructed without revealing b_ij
//
// Parameters:
//   - params: System parameters (g, h, q, p)
//   - C_i: Bidder's commitment
//   - e_ij: The value being proven
//   - proof: The ZK proof to verify
//   - j: Bit position (used in hash to prevent replay attacks)
//
// Returns: true if proof is valid, false otherwise
func VerifyZKProofEij(params *SystemParams, C_i *BidCommitment, e_ij *big.Int, proof *ZKProofEij, j int) bool {
	// Step 1: Verify that c1 + c2 = H(g, h, C_i, e_ij, a1, a2, j)
	expectedChallenge := computeChallenge(params, C_i.C, e_ij, proof.A1, proof.A2, j)
	sumC := new(big.Int).Add(proof.C1, proof.C2)
	sumC.Mod(sumC, params.Q)

	if sumC.Cmp(expectedChallenge) != 0 {
		return false // Challenge sum doesn't match
	}

	// Step 2: Verify first equation: g^{z1} * h^{z2} = a1 * e_ij^{c1}
	// This equation uses responses (z1, z2)
	// Left side: g^{z1} * h^{z2}
	leftSide1 := new(big.Int).Exp(params.G, proof.Z1, params.P)
	temp := new(big.Int).Exp(params.H, proof.Z2, params.P)
	leftSide1.Mul(leftSide1, temp)
	leftSide1.Mod(leftSide1, params.P)

	// Right side: a1 * e_ij^{c1}
	rightSide1 := new(big.Int).Exp(e_ij, proof.C1, params.P)
	rightSide1.Mul(rightSide1, proof.A1)
	rightSide1.Mod(rightSide1, params.P)

	if leftSide1.Cmp(rightSide1) != 0 {
		return false // First equation doesn't hold
	}

	// Step 3: Verify second equation: g^{w} * h^{v} = a2 * (e_ij / g)^{c2}
	// This equation uses responses (w, v)
	// Left side: g^{w} * h^{v}
	leftSide2 := new(big.Int).Exp(params.G, proof.W, params.P)
	temp = new(big.Int).Exp(params.H, proof.V, params.P)
	leftSide2.Mul(leftSide2, temp)
	leftSide2.Mod(leftSide2, params.P)

	// Right side: a2 * (e_ij / g)^{c2}
	// First compute e_ij / g = e_ij * g^{-1}
	gInv := new(big.Int).ModInverse(params.G, params.P)
	eijDivG := new(big.Int).Mul(e_ij, gInv)
	eijDivG.Mod(eijDivG, params.P)

	// Then compute (e_ij / g)^{c2}
	rightSide2 := new(big.Int).Exp(eijDivG, proof.C2, params.P)
	rightSide2.Mul(rightSide2, proof.A2)
	rightSide2.Mod(rightSide2, params.P)

	if leftSide2.Cmp(rightSide2) != 0 {
		return false // Second equation doesn't hold
	}

	// All checks passed - proof is valid!
	return true
}

// computeChallenge computes the Fiat-Shamir challenge
// c = H(g, h, C_i, e_ij, a1, a2, j)
// This binds the proof to the specific context and prevents replay attacks
func computeChallenge(params *SystemParams, C_i, e_ij, a1, a2 *big.Int, j int) *big.Int {
	hasher := sha256.New()

	// Hash all public parameters and values
	hasher.Write(params.G.Bytes())
	hasher.Write(params.H.Bytes())
	hasher.Write(C_i.Bytes())
	hasher.Write(e_ij.Bytes())
	hasher.Write(a1.Bytes())
	hasher.Write(a2.Bytes())
	hasher.Write([]byte{byte(j)}) // Include bit position j

	hashBytes := hasher.Sum(nil)

	// Convert hash to big.Int and reduce modulo q
	challenge := new(big.Int).SetBytes(hashBytes)
	challenge.Mod(challenge, params.Q)

	return challenge
}

// randBigInt generates a random big integer in [0, max)
func randBigInt(max *big.Int) (*big.Int, error) {
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Custom errors
var (
	ErrInvalidBitValue = &ZKError{msg: "bit value must be 0 or 1"}
)

type ZKError struct {
	msg string
}

func (e *ZKError) Error() string {
	return "ZK Proof Error: " + e.msg
}
