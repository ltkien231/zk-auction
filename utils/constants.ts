import { bls12_381 } from "@noble/curves/bls12-381.js";

export const L = 16; // bid bit-length
export const N = 4;  // number of bidders

export const CURVE = bls12_381;
export const Fr    = bls12_381.fields.Fr; // scalar field mod r (~255-bit prime)
export const Fp    = bls12_381.fields.Fp; // base field mod p (381-bit prime)

// Standard BLS12-381 G1 generator
export const G_POINT = bls12_381.G1.Point.BASE;

// Additive identity (point at infinity)
export const G_ZERO  = bls12_381.G1.Point.ZERO;

// H = hash-to-curve("SBRAC_H") — verifiably random, no known discrete log w.r.t. G
export const H_POINT = bls12_381.G1.hashToCurve(new TextEncoder().encode("SBRAC_H"));
