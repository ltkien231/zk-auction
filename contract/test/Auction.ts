import { time, loadFixture } from "@nomicfoundation/hardhat-toolbox-viem/network-helpers";
import { expect } from "chai";
import hre from "hardhat";
import { getAddress, parseEther } from "viem";
import { Bidder } from "../utils";

describe("Auction", function () {
  const DEPOSIT = parseEther("1");

  async function deployAuctionFixture() {
    // Contracts are deployed using the first signer/account by default
    const [purchaser, bidder1, bidder2, bidder3, bidder4] = await hre.viem.getWalletClients();
    const bidderWallets = [bidder1, bidder2, bidder3, bidder4];
    const biddersAddress = bidderWallets.map((b) => getAddress(b.account.address));
    const auction = await hre.viem.deployContract("Auction", [biddersAddress], {
      value: DEPOSIT,
    });

    const publicClient = await hre.viem.getPublicClient();

    return {
      auction,
      purchaser,
      bidderWallets,
      DEPOSIT,
      publicClient,
    };
  }

  describe("Deployment", function () {
    it("Should set the right unlockTime", async function () {
      const { auction, bidderWallets, purchaser } = await loadFixture(deployAuctionFixture);

      expect(await auction.read.purchaser()).to.equal(getAddress(purchaser.account.address));
      expect(await auction.read.n()).to.equal(BigInt(bidderWallets.length));
    });
  });

  describe("Auction", function () {
    const bids = [11, 10, 12, 100];
    const bidders = bids.map((bid, i) => new Bidder(i, bid));

    it("Should add bidders successfully", async function () {
      const { auction, bidderWallets, purchaser, publicClient } = await loadFixture(deployAuctionFixture);

      // Step 2: Add bidders
      for (let i = 0; i < bidders.length; i++) {
        const b = bidders[i];
        await auction.write.addBidder([b.commitment, b.pubX, b.pubS], {
          account: bidderWallets[i].account,
          value: parseEther("1"),
        });
      }

      const firstCommitment = await auction.read.commitments([0n]);
      expect(firstCommitment).to.equal(bidders[0].commitment);

      const publicKeyX_33 = await auction.read.publicKeysX([3n, 3n]);
      expect(publicKeyX_33).to.equal(bidders[3].pubX[3]);
    });

    async function deployAuctionFixture2() {
      // Contracts are deployed using the first signer/account by default
      const [purchaser, bidder1, bidder2, bidder3, bidder4] = await hre.viem.getWalletClients();
      const bidderWallets = [bidder1, bidder2, bidder3, bidder4];
      const biddersAddress = bidderWallets.map((b) => getAddress(b.account.address));
      const auction = await hre.viem.deployContract("Auction", [biddersAddress], {
        value: DEPOSIT,
      });

      for (let i = 0; i < bidders.length; i++) {
        const b = bidders[i];
        await auction.write.addBidder([b.commitment, b.pubX, b.pubS], {
          account: bidderWallets[i].account,
          value: parseEther("1"),
        });
      }

      const publicClient = await hre.viem.getPublicClient();

      return {
        auction,
        purchaser,
        bidderWallets,
        DEPOSIT,
        publicClient,
      };
    }

    it("Should add bidders successfully", async function () {
      const { auction, bidderWallets, purchaser, publicClient } = await loadFixture(deployAuctionFixture2);

      // Step 3: Set clearing price
      const minBid = Math.min(...bids);
      await auction.write.setClearingPrice([BigInt(minBid)]);

      // Step 4: Declare winner
      const winnerIndex = bids.indexOf(minBid);
      const winnerBidder = bidders[winnerIndex];
      await auction.write.declareWinner([BigInt(winnerBidder.salt)], {
        account: bidderWallets[winnerIndex].account,
      });
    });

    it("Should complete full auction flow", async function () {
      const { auction, bidderWallets, purchaser, publicClient } = await loadFixture(deployAuctionFixture2);

      // Step 3: Set clearing price
      const minBid = Math.min(...bids);
      await auction.write.setClearingPrice([BigInt(minBid)]);

      // Step 4: Declare winner
      const winnerIndex = bids.indexOf(minBid);
      const winnerBidder = bidders[winnerIndex];
      await auction.write.declareWinner([BigInt(winnerBidder.salt)], {
        account: bidderWallets[winnerIndex].account,
      });

      // Step 5: Withdraw funds
      const bidder1InitialBalance = await publicClient.getBalance({
        address: bidderWallets[0].account.address,
      });
      await auction.write.refundLosers({
        value: BigInt(minBid),
      });
      const bidder1FinalBalance = await publicClient.getBalance({
        address: bidderWallets[0].account.address,
      });
      expect(bidder1FinalBalance).to.equal(bidder1InitialBalance + DEPOSIT);

      // Finalize auction
      const winnerInitialBalance = await publicClient.getBalance({
        address: bidderWallets[winnerIndex].account.address,
      });
      await auction.write.finalize();
      const winnerFinalBalance = await publicClient.getBalance({
        address: bidderWallets[winnerIndex].account.address,
      });
      expect(winnerFinalBalance).to.equal(winnerInitialBalance + DEPOSIT + BigInt(minBid));
    });
  });
});
