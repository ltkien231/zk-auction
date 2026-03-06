// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

import "./BLS12381.sol";

/**
 * @title SBRAC - Sealed-Bid Reverse Auction Contract (BLS12-381 ECC)
 * @notice Implementation of sealed-bid reverse auction based on SBRAC paper.
 *         Uses BLS12-381 G1 via EIP-2537 (Prague) precompiles.
 *         ZK proof verification is omitted; core auction logic only.
 */
contract Auction {
    using BLS12381 for BLS12381.G1Point;

    // ============ Constants ============

    uint16 public constant BIT_LENGTH = 16;

    // ============ Public Parameters ============

    /// @notice Standard BLS12-381 G1 generator
    BLS12381.G1Point public G_POINT;
    /// @notice Second generator H = MAP_FP_TO_G1(keccak256("SBRAC_H")), no known dlog
    BLS12381.G1Point public H_POINT;

    // ============ State Variables ============

    address public purchaser;

    address[] public whitelist;
    mapping(address => bool) public whitelisted;
    uint256 public immutable N;

    address[] public joinedList;
    mapping(address => bool) public joined;
    mapping(address => uint256) public bidderIndex;

    uint256 public deposit;
    bool public auctionEnded;
    bool public isRefunded;

    address public winner;
    uint8[] public clearingPriceBits;
    uint256 public clearingPrice;

    /// @dev Pedersen commitments: C_i = bid_i * G + r_i * H
    mapping(uint256 => BLS12381.G1Point) public commitments;
    /// @dev AV-net round-1 public keys X_ij = x_ij * G, one per bidder per bit
    mapping(uint256 => BLS12381.G1Point[]) public publicXs;
    /// @dev AV-net round-1 public keys S_ij = s_ij * H, one per bidder per bit
    mapping(uint256 => BLS12381.G1Point[]) public publicSs;
    /// @dev Running sum of bit commitments per bit position (AV-net round-2 aggregate)
    mapping(uint256 => BLS12381.G1Point) public bitCommitSums;
    mapping(uint256 => uint256) public bitCommitCounts;

    // ============ Modifiers ============

    modifier onlyPurchaser() {
        require(msg.sender == purchaser, "Only purchaser can call");
        _;
    }

    modifier onlyBidder() {
        require(joined[msg.sender], "Not a registered bidder");
        _;
    }

    modifier notEnded() {
        require(!auctionEnded, "Auction already ended");
        _;
    }

    // ============ Phase 1: Constructor ============

    /**
     * @notice Deploy the auction contract.
     * @param _whitelist   Addresses allowed to participate as bidders.
     * @param _gPoint      BLS12-381 G1 generator (128 bytes as four bytes32).
     * @param _hPoint      Second generator H (128 bytes as four bytes32).
     */
    constructor(
        address[] memory _whitelist,
        BLS12381.G1Point memory _gPoint,
        BLS12381.G1Point memory _hPoint
    ) payable {
        require(msg.value > 0, "Purchaser must deposit");

        purchaser = msg.sender;
        deposit   = msg.value;
        N         = _whitelist.length;
        whitelist = _whitelist;
        G_POINT   = _gPoint;
        H_POINT   = _hPoint;

        for (uint256 i = 0; i < _whitelist.length; i++) {
            whitelisted[_whitelist[i]] = true;
        }
        // bitCommitSums default to (0,0,0,0) = point at infinity — correct identity
    }

    // ============ Phase 2: Add Bidders ============

    /**
     * @notice Register as a bidder and submit Pedersen commitment + AV-net public keys.
     * @param _commitment  C = bid*G + r*H  (G1 point)
     * @param _publicXs    X_ij = x_ij * G  (one G1 point per bit position)
     * @param _publicSs    S_ij = s_ij * H  (one G1 point per bit position)
     */
    function addBidder(
        BLS12381.G1Point calldata _commitment,
        BLS12381.G1Point[] calldata _publicXs,
        BLS12381.G1Point[] calldata _publicSs
    ) external payable {
        require(whitelisted[msg.sender], "Not whitelisted");
        require(!joined[msg.sender], "Already registered");
        require(msg.value == deposit, "Must match purchaser deposit");
        require(
            _publicXs.length == BIT_LENGTH && _publicSs.length == BIT_LENGTH,
            "Wrong number of public keys"
        );

        uint256 bid = joinedList.length;
        bidderIndex[msg.sender] = bid;
        joined[msg.sender]      = true;
        joinedList.push(msg.sender);

        commitments[bid] = _commitment;
        for (uint256 j = 0; j < BIT_LENGTH; j++) {
            publicXs[bid].push(_publicXs[j]);
            publicSs[bid].push(_publicSs[j]);
        }
    }

    // ============ Phase 3: Calculate Clearing Price ============

    /**
     * @notice Submit the AV-net round-2 bit commitment for a given bit position.
     *         Winners (bit=0) submit s_j * T_i; losers (bit=1) submit x_j * T_i.
     *         When all N bidders have submitted, the sum reveals the clearing price bit:
     *           isInfinity(sum) => all bits are 1 => clearing price bit = 1
     *           !isInfinity(sum) => at least one bit is 0 => clearing price bit = 0
     * @param _bitPosition  Bit index (0 = MSB).
     * @param _bitCommit    The AV-net cryptogram (G1 point).
     */
    function submitBitCommitment(
        uint256 _bitPosition,
        BLS12381.G1Point calldata _bitCommit
    ) external onlyBidder notEnded {
        // TODO: require bidderId has not submitted for bitPosition yet
        // TODO: check if we are in bitPosition phase
        require(_bitPosition < BIT_LENGTH, "Invalid bit position");

        BLS12381.G1Point memory currentSum = bitCommitSums[_bitPosition];
        bitCommitSums[_bitPosition] = BLS12381.add(currentSum, _bitCommit);
        bitCommitCounts[_bitPosition] += 1;

        if (bitCommitCounts[_bitPosition] == N) {
            BLS12381.G1Point memory finalSum = bitCommitSums[_bitPosition];
            if (BLS12381.isInfinity(finalSum)) {
                clearingPriceBits.push(1);
            } else {
                clearingPriceBits.push(0);
            }
            if (clearingPriceBits.length == BIT_LENGTH) {
                clearingPrice = _bitsToPrice();
            }
        }
    }

    // ============ Phase 4: Declare Winner ============

    /**
     * @notice Winner reveals their bid randomness to prove they hold the lowest bid.
     * @param _randomness  The salt r used in the Pedersen commitment C = bid*G + r*H.
     */
    function declareWinner(uint256 _randomness) external onlyBidder notEnded {
        require(clearingPriceBits.length == BIT_LENGTH, "Clearing price not determined");
        require(winner == address(0), "Winner already declared");

        uint256 bid = bidderIndex[msg.sender];
        BLS12381.G1Point memory stored = commitments[bid];

        BLS12381.G1Point memory g = G_POINT;
        BLS12381.G1Point memory h = H_POINT;
        BLS12381.G1Point memory computed = BLS12381.add(
            BLS12381.scalarMul(g, clearingPrice),
            BLS12381.scalarMul(h, _randomness)
        );

        require(BLS12381.eq(stored, computed), "Commitment mismatch");
        winner = msg.sender;
    }

    // ============ Phase 5: Refund Losers ============

    /**
     * @notice Purchaser sends the clearing price and refunds deposits to losing bidders.
     */
    function refundLosers() external payable onlyPurchaser notEnded{
        require(winner != address(0), "Winner not declared");
        require(!isRefunded, "Already refunded");
        require(msg.value == clearingPrice, "Must send clearing price");

        isRefunded = true;
        for (uint256 i = 0; i < joinedList.length; i++) {
            address bidder = joinedList[i];
            if (bidder != winner) {
                (bool ok,) = bidder.call{value: deposit}("");
                require(ok, "Refund failed");
            }
        }
    }

    // ============ Phase 6: Finalize ============

    /**
     * @notice Finalize the auction: return purchaser deposit and pay winner.
     */
    function finalize() external onlyPurchaser notEnded{
        require(winner != address(0), "Winner not declared");
        require(isRefunded, "Losers not refunded");

        auctionEnded = true;

        (bool ok1,) = purchaser.call{value: deposit}("");
        require(ok1, "Purchaser refund failed");

        (bool ok2,) = winner.call{value: deposit + clearingPrice}("");
        require(ok2, "Winner payment failed");
    }

    // ============ Views ============

    /**
     * @notice Returns all bidders' AV-net X public keys as a 2D array.
     *         Used off-chain to compute tally keys T_i.
     */
    function getPublicXs() external view returns (BLS12381.G1Point[][] memory) {
        uint256 total = joinedList.length;
        BLS12381.G1Point[][] memory all = new BLS12381.G1Point[][](total);
        for (uint256 i = 0; i < total; i++) {
            all[i] = publicXs[i];
        }
        return all;
    }

    // ============ Internal ============

    function _bitsToPrice() private view returns (uint256 price) {
        uint256 len = clearingPriceBits.length;
        for (uint256 j = 0; j < len; j++) {
            if (clearingPriceBits[j] == 1) {
                price += (1 << (len - 1 - j));
            }
        }
    }
}
