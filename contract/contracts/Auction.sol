// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

/**
 * @title SBRAC - Sealed-Bid Reverse Auction Contract
 * @notice Implementation of sealed-bid reverse auction based on SBRAC paper
 * @dev ZK proofs are omitted for simplicity, only core auction logic is implemented
 */
contract Auction {
    // ============ Constants ============
    
    uint16 public constant BIT_LENGTH = 32; 
    
    uint256 public constant P = 2039; 
    uint256 public constant Q = 1019; 
    uint256 public constant G = 9; 
    uint256 public constant H = 461;  
 
 
    // ============ State Variables ============
    
    address public purchaser; 

    address[] public whitelist; 
    mapping(address bidder => bool isWhitelisted) public whitelisted;
    uint256 public immutable N;

    address[] public joinedList; 
    mapping(address bidder => bool isJoined) public joined;
    mapping(address bidder => uint256 bidderId) public bidderIndex; 

    uint256 public deposit;
    bool public auctionEnded = false; 
    bool public isRefunded = false;
    
    address public winner; 
    uint8[] public clearingPriceBits; 
    uint256 public clearingPrice; 

    mapping(uint256 bidderId => uint256 commitment) public commitments;
    mapping(uint256 bidderId => uint256[] X) public publicXs;
    mapping(uint256 bidderId => uint256[] S) public publicSs;
    mapping(uint256 bidderId => mapping(uint256 bitPosition => uint256 bitCommitment)) public bitCommits;
    mapping(uint256 bitPosition => uint256 bitCommitCount) public bitCommitCounts;
    mapping(uint256 bitPosition => uint256 bitCommitProduct) public bitCommitProds;

    // ============ Events ============
    
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
     * @notice Deploy the auction contract
     */
    constructor(address[] memory _whitelist) payable {
        require(msg.value > 0, "Purchaser must deposit");
        
        purchaser = msg.sender;
        deposit = msg.value;
        N = _whitelist.length;
        whitelist = _whitelist;
        for (uint256 i = 0; i < N; i++) {
            whitelisted[_whitelist[i]] = true;
        }
        for (uint256 i = 0; i < BIT_LENGTH; i++) {
            bitCommitProds[i] = 1;
        }
    }
    
    // ============ Phase 2: Add Bidders & Submit Commitments ============
    
    /**
     * @notice Register as a bidder and submit commitments
     * @param _commitment Commitment C for the bidder
     * @param _publicXs Array of public keys X_ij
     * @param _publicSs Array of public keys S_ij
     */
    function addBidder(
        uint256 _commitment,
        uint256[] calldata _publicXs,
        uint256[] calldata _publicSs
    ) external payable notEnded {
        require(whitelisted[msg.sender], "Not whitelisted");
        require(!joined[msg.sender], "Already registered");
        require(msg.value == deposit, "Must deposit to participate");
        require(_publicXs.length == BIT_LENGTH && _publicSs.length == BIT_LENGTH, "Invalid publicKeys length");

        // Register bidder
        uint256 BID = joinedList.length;
        bidderIndex[msg.sender] = BID;

        whitelisted[msg.sender] = true;
        joined[msg.sender] = true;
        joinedList.push(msg.sender);

        // Store commitments and public keys
        commitments[BID] = _commitment;
        publicXs[BID] = _publicXs;
        publicSs[BID] = _publicSs;            
    }

    // ============ Phase 3: Verify Winner ============
    /**
     * @notice Purchaser sets the clearing price after verifying bids
     * @param _clearingPrice The final clearing price of the auction
     */
    function setClearingPrice(uint256 _clearingPrice) external onlyPurchaser notEnded {
        require(winner == address(0), "Winner already declared");
        clearingPrice = _clearingPrice;
    }

    function submitBitCommitment(
        uint256 bitPosition,
        uint256 bitCommitment
    ) external onlyBidder notEnded {
        // TODO: require bidderId has not submitted for bitPosition yet
        // TODO: check if we are in bitPosition phase
        uint256 BID = bidderIndex[msg.sender];
        bitCommits[BID][bitPosition] = bitCommitment;
        bitCommitProds[bitPosition] = (bitCommitProds[bitPosition] * bitCommitment) % P;
        
        bitCommitCounts[bitPosition] += 1;
        
        if (bitCommitCounts[bitPosition] == N) {
            if (bitCommitProds[bitPosition] == 1){
                clearingPriceBits.push(1);
            } else {
                clearingPriceBits.push(0);
            }
            if (clearingPriceBits.length == BIT_LENGTH) {
                clearingPrice = clearingPriceBitsToClearingPrice();
            }
        }
    }
    
    // ============ Phase 4: Verify Winner ============
    
    /**
     * @notice Winner declares themselves by revealing their bid
     * @param _randomness The randomness used in commitment
     */
    function declareWinner(
        uint256 _randomness
    ) external onlyBidder notEnded {
        require(winner == address(0), "Winner already declared");

        uint256 BID = bidderIndex[msg.sender];
        uint256 bidCommitment = commitments[BID];
        
        require(
            bidCommitment == pedersenCommit(clearingPrice, _randomness),
            "Invalid commitment - bid doesn't match clearing price"
        );
        
        winner = msg.sender;
        
    }
    
    // ============ Phase 5: Refund Deposits ============
    
    /**
     * @notice Refund deposits to losing bidders
     */
    function refundLosers() external payable onlyPurchaser notEnded {
        require(winner != address(0), "Winner not declared yet");
        require(!isRefunded, "Deposits already refunded");
        require(msg.value == clearingPrice, "Must send clearing price");
        isRefunded = true;
        for (uint256 i = 0; i < joinedList.length; i++) {
            address bidder = joinedList[i];
            if (bidder != winner) {                
                (bool success, ) = bidder.call{value: deposit}("");
                require(success, "Refund failed");
                
            }
        }
    }
    
    // ============ Phase 6: Finalize Auction ============
    
    /**
     * @notice Finalize auction and pay winner
     */
    function finalize() external onlyPurchaser notEnded {
        require(winner != address(0), "Winner not declared");
        require(isRefunded, "Loser deposits not refunded yet");
        require(!auctionEnded, "Auction already finalized");
        auctionEnded = true;
        
        // Transfer payments
        (bool success1, ) = purchaser.call{value: deposit}("");
        require(success1, "Purchaser refund failed");
        
        (bool success2, ) = winner.call{value: deposit + clearingPrice}("");
        require(success2, "Winner payment failed");        
    }

    function modPow(uint256 base, uint256 exp, uint256 mod) private pure returns (uint256) {
        uint256 result = 1;
        base = base % mod;
        while (exp > 0) {
            if (exp % 2 == 1) {
                result = (result * base) % mod;
            }
            base = (base * base) % mod;
            exp = exp >> 1;
        }
        return result;
    }

    function pedersenCommit(uint256 bid, uint256 randomness) public pure returns (uint256) {
        require(bid < Q, "Bid must be smaller than Q");
        require(randomness < Q, "Randomness must be smaller than Q"); 
        uint256 term1 = modPow(G, bid, P);
        uint256 term2 = modPow(H, randomness, P);
        return (term1 * term2) % P;
    }

    function getClearingPriceBits() external view returns (string memory) {
        return string(abi.encodePacked(clearingPriceBits));
    }

    function clearingPriceBitsToClearingPrice() private view returns (uint256) {
        uint256 price = 0;
        for (uint256 j = 0; j < clearingPriceBits.length; j++) {
            if (clearingPriceBits[j] == 1) {
                price += (1 << (clearingPriceBits.length - 1 - j));
            }
        }
        return price;
    }

    function getPublicXs() external view returns (uint256[][] memory) {
        uint256 totalBidders = joinedList.length;
        uint256[][] memory allPublicXs = new uint256[][](totalBidders);
        for (uint256 i = 0; i < totalBidders; i++) {
            allPublicXs[i] = publicXs[i];
        }
        return allPublicXs;
    }
}