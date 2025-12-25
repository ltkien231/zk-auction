package bidreveal

import (
	"math/big"
	"sbrac-auction/utils"
)

// SystemParams contains the public parameters for the auction system
type SystemParams struct {
	G *big.Int // Generator g of the cyclic group
	P *big.Int // Prime modulus
	Q *big.Int // Order of the group (prime)
}

type PrivateBitPair struct {
	X *big.Int // Random secret x_ij
	S *big.Int // Random secret s_ij
}

var systemParams *SystemParams

func init() {
	p := big.NewInt(23) // Prime
	q := big.NewInt(11) // Prime order where q | (p-1)
	g := big.NewInt(2)  // Generator

	systemParams = &SystemParams{
		P: p,
		Q: q,
		G: g,
	}
}

func DetermineClearingPrice(bids []int, bitLength int) int {
	bidders := make([]*Bidder, len(bids))
	for i, bid := range bids {
		bidders[i] = NewBidder(bid, i, bitLength, len(bids))
	}

	n := len(bids)
	if n == 0 {
		return 0
	}
	l := bitLength

	binaryBids := make([][]int, n)
	for i := 0; i < n; i++ {
		binaryBids[i] = utils.IntToBits(bids[i], l)
	}

	// bidders generate their private bit pairs
	privateBitPairs := make([][]PrivateBitPair, n)
	publicBitPairs := make([][]PrivateBitPair, n)
	for i := 0; i < n; i++ {
		privateBitPairs[i] = make([]PrivateBitPair, l)
		publicBitPairs[i] = make([]PrivateBitPair, l)
		for j := 0; j < l; j++ {
			// x_ij and s_ij are exponents, so they must be in Z_Q
			x := utils.RandBigInt(systemParams.Q)
			s := utils.RandBigInt(systemParams.Q)
			privateBitPairs[i][j] = PrivateBitPair{
				X: x,
				S: s,
			}
			publicBitPairs[i][j] = PrivateBitPair{
				X: new(big.Int).Exp(systemParams.G, x, systemParams.P),
				S: new(big.Int).Exp(systemParams.G, s, systemParams.P),
			}
		}
	}

	// Compute product of all X_kj for each bit position j
	// T_ij = (Product of all X_kj where k != i) = (Product of all X_kj) / X_ij
	totalProdX := make([]*big.Int, l)
	for j := 0; j < l; j++ {
		prod := big.NewInt(1)
		for i := 0; i < n; i++ {
			prod = utils.MulMod(prod, publicBitPairs[i][j].X, systemParams.P)
		}
		totalProdX[j] = prod
	}

	// T_ij = (Product of all X_kj) / X_ij
	tijs := make([][]*big.Int, n)
	for i := 0; i < n; i++ {
		tijs[i] = make([]*big.Int, l)
		for j := 0; j < l; j++ {
			// T_ij = totalProdX[j] / X_ij
			t_ij := utils.DivMod(totalProdX[j], publicBitPairs[i][j].X, systemParams.P)
			tijs[i][j] = t_ij
		}
	}

	// determine clearing price bits
	isLostBidder := make([]bool, n)
	clearingPriceBits := make([]int, l)
	for j := 0; j < l; j++ {
		hasZero := HasZeroAtBitPosition(tijs, isLostBidder, binaryBids, privateBitPairs, j)
		if hasZero {
			clearingPriceBits[j] = 0
		} else {
			clearingPriceBits[j] = 1
		}
	}

	return utils.BitsToInt(clearingPriceBits)
}

func HasZeroAtBitPosition(tijs [][]*big.Int, isLostBidder []bool, binaryBids [][]int, privateBitPairs [][]PrivateBitPair, j int) bool {
	n := len(binaryBids)
	if n == 0 {
		return false
	}

	eProduct := big.NewInt(1)

	for i := 0; i < n; i++ {
		b_ij := binaryBids[i][j]
		x_ij, s_ij := privateBitPairs[i][j].X, privateBitPairs[i][j].S

		e_ij := big.NewInt(0)
		if b_ij == 0 && !isLostBidder[i] {
			e_ij = new(big.Int).Exp(tijs[i][j], s_ij, systemParams.P)
		} else {
			e_ij = new(big.Int).Exp(tijs[i][j], x_ij, systemParams.P)
		}

		eProduct = utils.MulMod(eProduct, e_ij, systemParams.P)
	}

	hasZero := false
	if eProduct.Cmp(big.NewInt(1)) == 0 {
		hasZero = true
		for i := 0; i < n; i++ {
			b_ij := binaryBids[i][j]
			if b_ij == 1 {
				isLostBidder[i] = true
			}
		}
	}

	return hasZero
}
