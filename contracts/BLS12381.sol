// SPDX-License-Identifier: UNLICENSED
pragma solidity ^0.8.28;

/**
 * @title BLS12381
 * @notice Wrappers around EIP-2537 (Prague) BLS12-381 G1 precompiles.
 *
 * G1 point encoding per EIP-2537 (128 bytes total):
 *   x coordinate: 64 bytes = 16 zero-padding bytes || 48-byte Fp element (big-endian)
 *   y coordinate: 64 bytes = 16 zero-padding bytes || 48-byte Fp element (big-endian)
 *
 * Stored as four bytes32 fields:
 *   x_a = bytes  0..31  (16 zero bytes + top 16 bytes of x)
 *   x_b = bytes 32..63  (bottom 32 bytes of x)
 *   y_a = bytes 64..95  (16 zero bytes + top 16 bytes of y)
 *   y_b = bytes 96..127 (bottom 32 bytes of y)
 */
library BLS12381 {
    address private constant G1ADD        = address(0x0b);
    address private constant G1MSM        = address(0x0c);
    address private constant MAP_FP_TO_G1 = address(0x10);

    struct G1Point {
        bytes32 x_a;
        bytes32 x_b;
        bytes32 y_a;
        bytes32 y_b;
    }

    /**
     * @notice Add two G1 points. Supports point at infinity (0,0,0,0).
     */
    function add(G1Point memory p, G1Point memory q) internal view returns (G1Point memory r) {
        bytes memory input = abi.encodePacked(p.x_a, p.x_b, p.y_a, p.y_b, q.x_a, q.x_b, q.y_a, q.y_b);
        (bool ok, bytes memory out) = G1ADD.staticcall(input);
        require(ok && out.length == 128, "BLS12_G1ADD failed");
        assembly {
            mstore(r,           mload(add(out, 32)))
            mstore(add(r, 32),  mload(add(out, 64)))
            mstore(add(r, 64),  mload(add(out, 96)))
            mstore(add(r, 96),  mload(add(out, 128)))
        }
    }

    /**
     * @notice Scalar multiplication: scalar * p.
     * Scalar is a uint256 encoded as 32-byte big-endian (EIP-2537 G1MSM format, k=1).
     */
    function scalarMul(G1Point memory p, uint256 scalar) internal view returns (G1Point memory r) {
        bytes memory input = abi.encodePacked(p.x_a, p.x_b, p.y_a, p.y_b, bytes32(scalar));
        (bool ok, bytes memory out) = G1MSM.staticcall(input);
        require(ok && out.length == 128, "BLS12_G1MSM failed");
        assembly {
            mstore(r,           mload(add(out, 32)))
            mstore(add(r, 32),  mload(add(out, 64)))
            mstore(add(r, 64),  mload(add(out, 96)))
            mstore(add(r, 96),  mload(add(out, 128)))
        }
    }

    /**
     * @notice Map a 64-byte Fp field element to a G1 point (hash-to-curve helper).
     * @param fp_a  High 32 bytes of the 64-byte Fp encoding (first 16 bytes are zero-padding).
     * @param fp_b  Low 32 bytes of the 64-byte Fp encoding.
     */
    function mapFpToG1(bytes32 fp_a, bytes32 fp_b) internal view returns (G1Point memory r) {
        bytes memory input = abi.encodePacked(fp_a, fp_b);
        (bool ok, bytes memory out) = MAP_FP_TO_G1.staticcall(input);
        require(ok && out.length == 128, "BLS12_MAP_FP_TO_G1 failed");
        assembly {
            mstore(r,           mload(add(out, 32)))
            mstore(add(r, 32),  mload(add(out, 64)))
            mstore(add(r, 64),  mload(add(out, 96)))
            mstore(add(r, 96),  mload(add(out, 128)))
        }
    }

    /**
     * @notice Returns true if p is the point at infinity (identity element).
     */
    function isInfinity(G1Point memory p) internal pure returns (bool) {
        return p.x_a == bytes32(0) && p.x_b == bytes32(0) &&
               p.y_a == bytes32(0) && p.y_b == bytes32(0);
    }

    /**
     * @notice Returns true if two G1 points are equal.
     */
    function eq(G1Point memory p, G1Point memory q) internal pure returns (bool) {
        return p.x_a == q.x_a && p.x_b == q.x_b &&
               p.y_a == q.y_a && p.y_b == q.y_b;
    }
}
