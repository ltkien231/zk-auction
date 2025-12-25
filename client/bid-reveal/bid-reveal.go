package bidreveal

import (
	"fmt"
	"math/big"
	"sbrac-auction/utils"
)

// SystemParams contains the public parameters for the auction system
type SystemParams struct {
	G *big.Int // Generator g of the cyclic group
	H *big.Int // Second generator h of the cyclic group
	P *big.Int // Prime modulus
	Q *big.Int // Order of the group (prime)
}

type BitPair struct {
	X *big.Int // Random secret x_ij
	S *big.Int // Random secret s_ij
}

type WhiteBoard struct {
	PubXs [][]*big.Int // Public X_ij values from all bidders
}

var systemParams *SystemParams

func init() {
	p := big.NewInt(2039) // Prime
	q := big.NewInt(1019) // Prime order where q | (p-1)
	g := big.NewInt(9)    // Generator
	h := big.NewInt(461)  // Second generator

	systemParams = &SystemParams{
		P: p,
		Q: q,
		G: g,
		H: h,
	}
}

func DetermineClearingPrice(bids []int, bitLength int) int {
	bidders := make([]*Bidder, len(bids))
	for i, bid := range bids {
		bidders[i] = NewBidder(bid, i, bitLength, len(bids))
	}
	fmt.Println("Bidders initialized:", bidders[0])

	n := len(bids)
	if n == 0 {
		return 0
	}
	l := bitLength

	PubXs := make([][]*big.Int, len(bidders))
	for i, bidder := range bidders {
		PubXs[i] = make([]*big.Int, bidder.L)
		for j := 0; j < bidder.L; j++ {
			PubXs[i][j] = bidder.PublicBitPairs[j].X
		}
	}

	for _, bidder := range bidders {
		bidder.ComputeTi(PubXs)
	}

	// determine clearing price bits
	clearingPriceBits := make([]int, l)
	for j := 0; j < l; j++ {
		hasZero := HasZeroAtBitPosition(bidders, j)
		fmt.Println("Clearing price bits:", j, hasZero)
		if hasZero {
			clearingPriceBits[j] = 0
		} else {
			clearingPriceBits[j] = 1
		}
	}
	return utils.BitsToInt(clearingPriceBits)
}

func HasZeroAtBitPosition(bidders []*Bidder, j int) bool {
	n := len(bidders)
	if n == 0 {
		return false
	}

	if j >= 12 {
		fmt.Println("Bit position at", j, bidders[0].BinaryBid[j], bidders[1].BinaryBid[j], bidders[2].BinaryBid[j])
	}

	eProduct := big.NewInt(1)

	for i := 0; i < n; i++ {
		b_ij := bidders[i].BinaryBid[j]
		x_ij, s_ij := bidders[i].PrivateBitPairs[j].X, bidders[i].PrivateBitPairs[j].S

		e_ij := big.NewInt(0)
		if b_ij == 0 && !bidders[i].IsLost {
			e_ij = new(big.Int).Exp(bidders[i].Ti[j], s_ij, systemParams.P)
		} else {
			e_ij = new(big.Int).Exp(bidders[i].Ti[j], x_ij, systemParams.P)
		}

		eProduct = utils.MulMod(eProduct, e_ij, systemParams.P)
	}

	hasZero := false
	if eProduct.Cmp(big.NewInt(1)) != 0 { // e != 1 means at least one b_ij = 0
		hasZero = true
		for i := 0; i < n; i++ {
			b_ij := bidders[i].BinaryBid[j]
			if b_ij == 1 {
				bidders[i].IsLost = true
			}
		}
	}

	return hasZero
}
