# Sealed-Bid Reverse Auction on Blockchain

## Overview

Implementation of a smart-contract-based, privacy-preserving sealed-bid reverse auction

**Auction type:**

- reverse auction: lowest bid wins
- purchaser procures, bidders are suppliers
- sealed-bid: bid is keep private

---

## Design Goals

The scheme simultaneously achieves:

1. **Decentralization** — no trusted auctioneer; smart contract replaces the auctioneer.
2. **Bid confidentiality** — bids are hidden via Pedersen commitments; only the winning bid is ever revealed.
3. **Privacy of losing bids** — even after the auction, losing bid values remain private as long as at least one honest bidder exists.
4. **Collusion resistance** — guaranteed under the DDH assumption; all-but-one collusion cannot expose the honest bidder's bid.
5. **Public verifiability** — all data is on-chain; anyone can verify correctness using ZK proofs.
6. **Fairness** — binding commitments prevent bid changes after submission.
7. **Non-repudiation** — immutable on-chain records.

---

## Cryptographic Primitives

| Primitive | Purpose |
|---|---|
| **Pedersen Commitment** `C = G^bid * H^r mod P` | Commit to bid without revealing it; binding under DLP |
| **NIZK (Schnorr-based, Fiat-Shamir)** | Prove knowledge of commitment secret; prove correct AV-net bit encoding |
| **Anonymous Veto Network (AV-net)** | Aggregate bits across all bidders without revealing individual bits |

**Group parameters (toy/dev values):**
- `P = 2039` (prime, multiplication modulus)
- `Q = 1019` (prime, exponent modulus, `P = 2Q + 1`)
- `G = 9`, `H = 461` (generators)
- `L = 16` (bid bit-length)

> **Note:** These are small parameters for local testing. Production requires cryptographically large primes.

---

## Protocol Phases

### Phase 1: Deploy Contract
- Purchaser deploys `Auction.sol` with the bidder whitelist and a required deposit.
- Contract is initialized with group parameters and `BIT_LENGTH`.

### Phase 2: Add Bidders (`addBidder`)
- Each whitelisted bidder calls `addBidder`, submitting:
  - `_commitment` — Pedersen commitment `C = G^bid * H^r mod P`
  - `_publicXs[L]` — per-bit AV-net public keys `X_ij = G^x_ij mod P`
  - `_publicSs[L]` — per-bit AV-net public keys `S_ij = H^s_ij mod P`
- Bidder pays the same deposit as the purchaser.
- Once submitted, a bid cannot be withdrawn.

### Phase 3: Calculate Clearing Price (`submitBitCommitment`)
- Conducted bit-by-bit from MSB to LSB over `L` rounds.
- In each round, every bidder submits a **bit commitment** (masked AV-net value for that bit position).
  - **Losers** (whose bit at this position is 1 while the minimum is 0) submit all-1 encodings for remaining rounds, hiding that they have lost.
- When all `N` bidders have submitted for a bit position, the contract multiplies all values mod `P`:
  - Product `== 1` → all bidders have a `1` at this bit → clearing price bit = `1`
  - Product `!= 1` → at least one bidder has a `0` → clearing price bit = `0`
- After all `L` bits, `clearingPrice` is assembled from `clearingPriceBits`.

### Phase 4: Declare Winner (`declareWinner`)
- The bidder whose bid equals the clearing price calls `declareWinner` with their commitment randomness `r`.
- Contract verifies: `commitments[BID] == G^clearingPrice * H^r mod P`.
- If valid, `winner` is set. (Forgery is computationally infeasible under DLP.)

### Phase 5: Refund Losers (`refundLosers`)
- Purchaser calls `refundLosers`, sending `clearingPrice` ETH.
- Contract returns deposits to all non-winning bidders.

### Phase 6: Finalize (`finalize`)
- Purchaser calls `finalize` after delivery.
- Contract pays winner `deposit + clearingPrice` and returns the purchaser's deposit.

---

## Project Structure

```
contracts/
  Auction.sol          # Main SBRAC smart contract (Solidity, Hardhat)

utils/
  constants.ts         # Group parameters: P, Q, G, H, L, N
  math.ts              # Modular arithmetic: modPow, pedersenCommit, modInv, etc.
  bidder.ts            # Bidder class: key generation, commitment, AV-net bit commitments
  index.ts             # Re-exports

test/
  Auction.ts           # Integration tests for the contract
  math.test.ts         # Unit tests for math utilities
```

---

## Bidder Off-Chain Computation (`Bidder` class)

Each bidder instantiates a `Bidder(id, bid)` which:
1. Generates a random `salt` (randomness `r`) and computes `commitment = pedersenCommit(bid, salt)`.
2. For each bit position `j ∈ [0, L)`:
   - Generates private/public AV-net keys: `(x_j, X_j = G^x_j)` and `(s_j, S_j = H^s_j)`.
   - Generates a Schnorr NIZK proof of knowledge of `x_j` and `s_j`.
3. After all bidders have joined (`computeBitCommitments(pubXs)`):
   - Computes the "tally key" `T_i = (∏_{k<i} X_k[j]) / (∏_{k>i} X_k[j]) mod P` for each bit.
   - Computes `bitZeroCommitment[j] = T_i^s_j mod P` (for bit = 0).
   - Computes `bitOneCommitment[j] = T_i^x_j mod P` (for bit = 1 / loser encoding).

---

## Key Invariants

- All arithmetic is modular: exponents mod `Q`, products mod `P`.
- `G` and `H` must be generators of a group of order `Q` where `P = 2Q + 1` (safe prime structure).
- `bid < Q` and `salt < Q` (enforced in both Solidity and TypeScript).
- `N` (number of bidders) must match `whitelist.length` passed to the constructor.
- AV-net correctness: product of all bit commitments `== 1` iff all bids have bit `1` at that position.

---

## Current Implementation Notes

- ZK proof verification is **omitted** in the smart contract (noted as TODO); the contract stores and uses proofs off-chain for now.
- `bitCommits` are submitted without enforcing per-bidder submission order or round gating (TODOs in `submitBitCommitment`).
- `BIT_LENGTH`, `P`, `Q`, `G`, `H` are compile-time constants in Solidity.
- The `intToBits` function in `math.ts` uses LSB-first order; `clearingPriceBitsToClearingPrice` in Solidity uses MSB-first — verify consistency in tests.
