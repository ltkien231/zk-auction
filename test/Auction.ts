import { loadFixture } from "@nomicfoundation/hardhat-toolbox-viem/network-helpers";
import { expect } from "chai";
import hre from "hardhat";
import { getAddress, parseEther } from "viem";
import { Bidder, G_POINT, H_POINT, L, N, pointToViem } from "../utils";

const G_VIEM = pointToViem(G_POINT);
const H_VIEM = pointToViem(H_POINT);

describe("Auction", function () {
  const DEPOSIT = parseEther("1");
  const bids    = [583, 324, 903, 785];
  const bidders = bids.map((bid, i) => new Bidder(i, bid));

  async function deployAuctionFixture() {
    const [purchaser, bidder1, bidder2, bidder3, bidder4] = await hre.viem.getWalletClients();
    const bidderWallets   = [bidder1, bidder2, bidder3, bidder4];
    const biddersAddress  = bidderWallets.map((b) => getAddress(b.account.address));

    const auction = await hre.viem.deployContract("Auction", [biddersAddress, G_VIEM, H_VIEM], {
      value: DEPOSIT,
    });

    const publicClient = await hre.viem.getPublicClient();
    return { auction, purchaser, bidderWallets, publicClient };
  }

  async function deployAndAddBiddersFixture() {
    const { auction, purchaser, bidderWallets, publicClient } = await loadFixture(deployAuctionFixture);

    for (let i = 0; i < bidders.length; i++) {
      const b = bidders[i];
      await auction.write.addBidder([b.commitment, b.pubX, b.pubS], {
        account: bidderWallets[i].account,
        value: DEPOSIT,
      });
    }

    return { auction, purchaser, bidderWallets, publicClient };
  }

  // ─── Deployment ────────────────────────────────────────────────────────────

  describe("Deployment", function () {
    it("sets purchaser and N correctly", async function () {
      const { auction, purchaser, bidderWallets } = await loadFixture(deployAuctionFixture);
      expect(await auction.read.purchaser()).to.equal(getAddress(purchaser.account.address));
      expect(await auction.read.N()).to.equal(BigInt(bidderWallets.length));
    });
  });

  // ─── Add Bidders ───────────────────────────────────────────────────────────

  describe("addBidder", function () {
    it("stores commitment and public keys correctly", async function () {
      const { auction } = await loadFixture(deployAndAddBiddersFixture);

      // Mapping auto-getters return flat arrays [x_a, x_b, y_a, y_b], not named objects.
      // Check bidder 0 commitment
      const stored = await auction.read.commitments([0n]) as unknown as readonly [`0x${string}`, `0x${string}`, `0x${string}`, `0x${string}`];
      const expected = bidders[0].commitment;
      expect(stored[0]).to.equal(expected.x_a);
      expect(stored[1]).to.equal(expected.x_b);
      expect(stored[2]).to.equal(expected.y_a);
      expect(stored[3]).to.equal(expected.y_b);

      // Check bidder 3, bit 3 public key X
      const storedX = await auction.read.publicXs([3n, 3n]) as unknown as readonly [`0x${string}`, `0x${string}`, `0x${string}`, `0x${string}`];
      const expectedX = bidders[3].pubX[3];
      expect(storedX[0]).to.equal(expectedX.x_a);
      expect(storedX[1]).to.equal(expectedX.x_b);
    });
  });

  // ─── Full Auction Flow ─────────────────────────────────────────────────────

  describe("Full auction flow", function () {
    it("determines correct clearing price and pays winner", async function () {
      const { auction, bidderWallets, publicClient } = await loadFixture(deployAndAddBiddersFixture);

      // Compute tally keys from on-chain public keys
      const allPubXs = await auction.read.getPublicXs();
      for (const bidder of bidders) {
        bidder.computeBitCommitments(allPubXs);
      }

      // Phase 3: submit bit commitments MSB → LSB
      for (let j = 0; j < L; j++) {
        for (const bidder of bidders) {
          const bitCommit =
            bidder.bidBinary[j] === 0 && !bidder.isLost
              ? bidder.bitZeroCommitments[j]
              : bidder.bitOneCommitments[j];

          await auction.write.submitBitCommitment([BigInt(j), bitCommit], {
            account: bidderWallets[bidder.id].account,
          });
        }

        const clearingPriceBit = await auction.read.clearingPriceBits([BigInt(j)]);
        if (clearingPriceBit === 0) {
          // Bidders with bit=1 at this position have lost
          for (const bidder of bidders) {
            if (bidder.bidBinary[j] === 1) bidder.isLost = true;
          }
        }
      }

      const clearingPrice = await auction.read.clearingPrice();
      const expectedMin   = BigInt(Math.min(...bids));
      console.log("clearing price:", clearingPrice, "expected:", expectedMin);
      expect(clearingPrice).to.equal(expectedMin);

      // Phase 4: declare winner
      const minBid      = Math.min(...bids);
      const winnerIndex = bids.indexOf(minBid);
      const winnerBidder = bidders[winnerIndex];

      await auction.write.declareWinner([winnerBidder.salt], {
        account: bidderWallets[winnerIndex].account,
      });

      // Phase 5: refund losers
      const nextIndex = (winnerIndex + 1) % N;
      const loserBalanceBefore = await publicClient.getBalance({
        address: bidderWallets[nextIndex].account.address,
      });

      await auction.write.refundLosers({ value: BigInt(minBid) });

      const loserBalanceAfter = await publicClient.getBalance({
        address: bidderWallets[nextIndex].account.address,
      });
      expect(loserBalanceAfter).to.equal(loserBalanceBefore + DEPOSIT);

      // Phase 6: finalize
      const winnerBalanceBefore = await publicClient.getBalance({
        address: bidderWallets[winnerIndex].account.address,
      });

      await auction.write.finalize();

      const winnerBalanceAfter = await publicClient.getBalance({
        address: bidderWallets[winnerIndex].account.address,
      });
      expect(winnerBalanceAfter).to.equal(winnerBalanceBefore + DEPOSIT + BigInt(minBid));
    });
  });
});
