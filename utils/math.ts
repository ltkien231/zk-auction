import { bls12_381 } from "@noble/curves/bls12-381.js";
import { G_POINT, H_POINT, G_ZERO, Fr } from "./constants";

// ─── Types ───────────────────────────────────────────────────────────────────

export type G1Point = typeof bls12_381.G1.Point.BASE;

/**
 * Viem-compatible G1Point representation.
 * Maps to Solidity: struct G1Point { bytes32 x_a; bytes32 x_b; bytes32 y_a; bytes32 y_b; }
 *
 * Each Fp coordinate is 48 bytes, encoded as 64 bytes (EIP-2537):
 *   [16 zero bytes || 48-byte Fp value big-endian]
 * Split into two bytes32:
 *   *_a = bytes  0..31  (16 zero bytes + top 16 bytes of Fp)
 *   *_b = bytes 32..63  (bottom 32 bytes of Fp)
 */
export type G1PointViem = {
  x_a: `0x${string}`;
  x_b: `0x${string}`;
  y_a: `0x${string}`;
  y_b: `0x${string}`;
};

// ─── Bit helpers ─────────────────────────────────────────────────────────────

/** Convert integer to bit array, LSB first. */
export function intToBits(n: number, width: number): number[] {
  const bits: number[] = [];
  for (let i = 0; i < width; i++) bits.push((n >> i) & 1);
  return bits;
}

/** Convert bit array (LSB first) back to integer. */
export function bitsToInt(bits: number[]): number {
  let n = 0;
  for (let i = 0; i < bits.length; i++) n |= bits[i] << i;
  return n;
}

/** Convert integer to bit array, MSB first (used by auction protocol). */
export function intToBitsMSB(n: number, width: number): number[] {
  return n.toString(2).padStart(width, "0").split("").map(Number);
}

// ─── Scalar helpers ───────────────────────────────────────────────────────────

/** Cryptographically random scalar in [1, r-1]. */
export function randomScalar(): bigint {
  const sk = bls12_381.utils.randomSecretKey();
  let hex = "";
  for (const b of sk) hex += b.toString(16).padStart(2, "0");
  return Fr.create(BigInt("0x" + hex));
}

// ─── EC point operations ──────────────────────────────────────────────────────

export const scalarMul = (p: G1Point, s: bigint): G1Point => p.multiply(Fr.create(s));

export const pointAdd = (p: G1Point, q: G1Point): G1Point => p.add(q);

export const pointSub = (p: G1Point, q: G1Point): G1Point => p.add(q.negate());

export const pointNeg = (p: G1Point): G1Point => p.negate();

/** Pedersen commitment: bid*G + r*H */
export function pedersenCommit(bid: bigint, r: bigint): G1Point {
  return scalarMul(G_POINT, bid).add(scalarMul(H_POINT, r));
}

// ─── Encoding: G1Point ↔ Viem ────────────────────────────────────────────────

/**
 * Encode a 48-byte Fp element as two bytes32 in EIP-2537 format:
 *   [16 zero bytes || 48-byte Fp big-endian] → (hi: bytes 0..31, lo: bytes 32..63)
 */
function fpToBytes32Pair(fp: bigint): { hi: `0x${string}`; lo: `0x${string}` } {
  const hex = fp.toString(16).padStart(96, "0"); // 48 bytes = 96 hex chars
  // 64-byte encoding: 32 zero hex chars + 96 hex chars of fp = 128 hex chars total
  // hi = bytes  0..31: "0".repeat(32) + first 32 hex chars of fp
  // lo = bytes 32..63: last 64 hex chars of fp
  return {
    hi: `0x${"0".repeat(32)}${hex.slice(0, 32)}`,
    lo: `0x${hex.slice(32)}`,
  };
}

/** Convert a G1 point to the Viem struct format for Solidity calls. */
export function pointToViem(p: G1Point): G1PointViem {
  const { x, y } = p.toAffine();
  const xp = fpToBytes32Pair(x);
  const yp = fpToBytes32Pair(y);
  return { x_a: xp.hi, x_b: xp.lo, y_a: yp.hi, y_b: yp.lo };
}

/** Decode a bytes32 pair (hi, lo) back to a 48-byte Fp element (bigint). */
function bytes32PairToFp(hi: `0x${string}`, lo: `0x${string}`): bigint {
  // hi = 64 hex chars: first 32 are zero-padding, next 32 are top 16 bytes of Fp
  // lo = 64 hex chars: bottom 32 bytes of Fp
  const hiHex = hi.replace(/^0x/, "").slice(32); // top 16 bytes of Fp
  const loHex = lo.replace(/^0x/, "");            // bottom 32 bytes of Fp
  return BigInt("0x" + hiHex + loHex);
}

/**
 * Convert a Viem G1Point back to a G1 projective point.
 * Accepts either a named object {x_a,x_b,y_a,y_b} (from explicit function returns)
 * or a plain 4-element array [x_a,x_b,y_a,y_b] (from mapping auto-getters).
 */
export function viemToPoint(v: G1PointViem | readonly [`0x${string}`, `0x${string}`, `0x${string}`, `0x${string}`]): G1Point {
  const isArr = Array.isArray(v);
  const x_a = isArr ? (v as any)[0] : (v as G1PointViem).x_a;
  const x_b = isArr ? (v as any)[1] : (v as G1PointViem).x_b;
  const y_a = isArr ? (v as any)[2] : (v as G1PointViem).y_a;
  const y_b = isArr ? (v as any)[3] : (v as G1PointViem).y_b;
  const x = bytes32PairToFp(x_a, x_b);
  const y = bytes32PairToFp(y_a, y_b);
  return bls12_381.G1.Point.fromAffine({ x, y });
}
