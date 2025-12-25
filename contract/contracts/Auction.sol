// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

/**
 * @title SBRAC - Sealed-Bid Reverse Auction Contract
 * @notice Implementation of sealed-bid reverse auction based on SBRAC paper
 * @dev ZK proofs are omitted for simplicity, only core auction logic is implemented
 */
contract Auction {
    // ============ Constants ============
    
    uint8 public constant BIT_LENGTH = 8; // Length of bid in bits
    
    // Pedersen commitment parameters
    uint256 public constant P = 2039; 
    uint256 public constant Q = 1019; 
    uint256 public constant G = 9;  // Small generator g
    uint256 public constant H = 461;  // Small generator h
 
    // ============ State Variables ============
    
    address public purchaser; // The buyer who deploys the contract

    address[] public whitelist; // whitelisted bidders
    mapping(address => bool) public whitelisted;
    uint256 public n; // number of bidders

    address[] public joinedList; // registered bidders
    mapping(address => bool) public joined;
    mapping(address => uint256) public bidderIndex; // bidder address to index

    uint256 public deposit;
    bool public auctionEnded = false; // Whether auction has finalized
    bool public isRefunded = false; // Whether deposits have been refunded
    
    address public winner; // The winning bidder
    uint256 public clearingPrice; // The final auction price
    

    mapping(uint256 => uint256) public commitments;
    mapping(uint256 => mapping(uint256 => uint256)) public publicKeysX;
    mapping(uint256 => mapping(uint256 => uint256)) public publicKeysS;

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
        n = _whitelist.length;
        whitelist = _whitelist;
        for (uint256 i = 0; i < n; i++) {
            whitelisted[_whitelist[i]] = true;
        }
    }
    
    // ============ Phase 2: Add Bidders & Submit Commitments ============
    
    /**
     * @notice Register as a bidder and submit commitments
     * @param _commitment Commitment C for the bidder
     * @param _publicKeysX Array of public keys X_ij
     * @param _publicKeysS Array of public keys S_ij
     */
    function addBidder(
        uint256 _commitment,
        uint256[] calldata _publicKeysX,
        uint256[] calldata _publicKeysS
    ) external payable notEnded {
        require(whitelisted[msg.sender], "Not whitelisted");
        require(!joined[msg.sender], "Already registered");
        require(msg.value == deposit, "Must deposit to participate");
        require(_publicKeysX.length == BIT_LENGTH && _publicKeysS.length == BIT_LENGTH, "Invalid publicKeys length");

        // Register bidder
        uint256 BID = joinedList.length;
        bidderIndex[msg.sender] = BID;

        whitelisted[msg.sender] = true;
        joined[msg.sender] = true;
        joinedList.push(msg.sender);

        // Store commitments and public keys
        for (uint256 j = 0; j < BIT_LENGTH; j++) {
            commitments[BID] = _commitment;
            publicKeysX[BID][j] = _publicKeysX[j];
            publicKeysS[BID][j] = _publicKeysS[j];            
        }
        
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

    function modPow(uint256 base, uint256 exp, uint256 mod) internal pure returns (uint256) {
        uint256 result = 1;
        base = base % mod;
        while (exp > 0) {
            if (exp % 2 == 1) {
                result = (result * base) % mod;
            }
            exp = exp >> 1;
            base = (base * base) % mod;
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
}