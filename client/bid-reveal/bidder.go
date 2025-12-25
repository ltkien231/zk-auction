package bidreveal

import (
	"math/big"
	"sbrac-auction/utils"
)

type Bidder struct {
	ID              int // Unique identifier for the bidder
	Bid             int // The bidder's bid
	privateBitPairs []PrivateBitPair
	publicBitPairs  []PrivateBitPair
	isLost          bool
	Ti              []*big.Int // T_ij values for each bit position
	l               int        // Bit length of the bid
	n               int        // Total number of bidders
}

func NewBidder(bid int, id int, bitLength int, n int) *Bidder {
	bidder := &Bidder{
		ID:              id,
		Bid:             bid,
		privateBitPairs: make([]PrivateBitPair, bitLength),
		publicBitPairs:  make([]PrivateBitPair, bitLength),
		isLost:          false,
		Ti:              make([]*big.Int, bitLength),
		l:               bitLength,
		n:               n,
	}

	for j := 0; j < bitLength; j++ {
		x := utils.RandBigInt(systemParams.Q)
		s := utils.RandBigInt(systemParams.Q)
		bidder.privateBitPairs[j] = PrivateBitPair{
			X: x,
			S: s,
		}
		bidder.publicBitPairs[j] = PrivateBitPair{
			X: new(big.Int).Exp(systemParams.G, x, systemParams.P),
			S: new(big.Int).Exp(systemParams.G, s, systemParams.P),
		}
	}

	return bidder
}

func (b *Bidder) ComputeTi(publicXs [][]*big.Int) {
	for j := 0; j < b.l; j++ {
		preProd := big.NewInt(1)
		for k := 0; k < b.ID; k++ {
			preProd = utils.MulMod(preProd, publicXs[k][j], systemParams.P)
		}
		postProd := big.NewInt(1)
		for k := b.ID + 1; k < b.n; k++ {
			postProd = utils.MulMod(postProd, publicXs[k][j], systemParams.P)
		}

		b.Ti[j] = utils.DivMod(preProd, postProd, systemParams.P)
	}
}
