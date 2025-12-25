package bidreveal

import (
	"encoding/json"
	"fmt"
	"math/big"
	"sbrac-auction/utils"
)

type Bidder struct {
	ID              int        `json:"id"`
	Bid             int        `json:"bid"`
	BinaryBid       []int      `json:"binary_bid"`
	PrivateBitPairs []BitPair  `json:"private_bit_pairs"`
	PublicBitPairs  []BitPair  `json:"public_bit_pairs"`
	IsLost          bool       `json:"is_lost"`
	Ti              []*big.Int `json:"ti"`
	L               int        `json:"l"`
	N               int        `json:"n"`
}

func NewBidder(bid int, id int, bitLength int, n int) *Bidder {
	bidder := &Bidder{
		ID:              id,
		Bid:             bid,
		BinaryBid:       utils.IntToBits(bid, bitLength),
		PrivateBitPairs: make([]BitPair, bitLength),
		PublicBitPairs:  make([]BitPair, bitLength),
		IsLost:          false,
		Ti:              make([]*big.Int, bitLength),
		L:               bitLength,
		N:               n,
	}

	for j := 0; j < bitLength; j++ {
		x := utils.RandBigInt(systemParams.Q)
		s := utils.RandBigInt(systemParams.Q)
		bidder.PrivateBitPairs[j] = BitPair{
			X: x,
			S: s,
		}
		bidder.PublicBitPairs[j] = BitPair{
			X: new(big.Int).Exp(systemParams.G, x, systemParams.P),
			S: new(big.Int).Exp(systemParams.G, s, systemParams.P),
		}
	}

	return bidder
}

func (b *Bidder) ComputeTi(publicXs [][]*big.Int) {
	for j := 0; j < b.L; j++ {
		preProd := big.NewInt(1)
		for k := 0; k < b.ID; k++ {
			preProd = utils.MulMod(preProd, publicXs[k][j], systemParams.P)
		}
		postProd := big.NewInt(1)
		for k := b.ID + 1; k < b.N; k++ {
			postProd = utils.MulMod(postProd, publicXs[k][j], systemParams.P)
		}

		b.Ti[j] = utils.DivMod(preProd, postProd, systemParams.P)
	}
}

func (b *Bidder) String() string {
	jsonByte, err := json.MarshalIndent(b, "", "    ")
	if err != nil {
		fmt.Println(err)
		return "" // DOTO: must string
	}

	return string(jsonByte)
}
