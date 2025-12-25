package bidreveal

import "testing"

func TestBitReveal(t *testing.T) {
	bids := []int{10, 11, 12}
	clearingPrice := DetermineClearingPrice(bids, 4)
	expectedClearingPrice := 10
	if clearingPrice != expectedClearingPrice {
		t.Errorf("Expected clearing price %d, got %d", expectedClearingPrice, clearingPrice)
	}
}

func TestAuction(t *testing.T) {
	testCases := []struct {
		bids                  []int
		bidLength             int
		expectedClearingPrice int
	}{
		{bids: []int{5, 7, 9}, bidLength: 4, expectedClearingPrice: 5},
		{bids: []int{159, 102, 890, 215}, bidLength: 10, expectedClearingPrice: 102},
		{bids: []int{8, 6, 7, 5}, bidLength: 4, expectedClearingPrice: 5},
		{bids: []int{42}, bidLength: 6, expectedClearingPrice: 42},
	}

	for _, tc := range testCases {
		clearingPrice := DetermineClearingPrice(tc.bids, tc.bidLength)
		if clearingPrice != tc.expectedClearingPrice {
			t.Errorf("For bids %v, expected clearing price %d, got %d",
				tc.bids, tc.expectedClearingPrice, clearingPrice)
		}
	}
}
