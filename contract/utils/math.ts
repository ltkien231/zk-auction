import { G, H, P, Q } from "./constants";

export function intToBits(n: number, width: number): number[] {
  const bits: number[] = [];
  for (let i = 0; i < width; i++) {
    bits.push((n >> i) & 1);
  }
  return bits;
}

export function bitsToInt(bits: number[]): number {
  let n = 0;
  for (let i = 0; i < bits.length; i++) {
    n |= bits[i] << i;
  }
  return n;
}

export function randomInt(max: number): number {
  return Math.floor(Math.random() * max);
}

function egcd(a: bigint, b: bigint): [bigint, bigint, bigint] {
  if (b === 0n) return [a, 1n, 0n];
  const [g, x1, y1] = egcd(b, a % b);
  return [g, y1, x1 - (a / b) * y1];
}

export function modInv(a: bigint, mod: bigint): bigint {
  const [g, x] = egcd(a, mod);
  if (g !== 1n) throw new Error("No modular inverse");
  return ((x % mod) + mod) % mod;
}

export function modDiv(a: bigint, b: bigint, mod: bigint): bigint {
  return (a * modInv(b, mod)) % mod;
}

export function modMul(a: bigint, b: bigint, mod: bigint): bigint {
  return (a * b) % mod;
}

export function modAdd(a: bigint, b: bigint, mod: bigint): bigint {
  return (a + b) % mod;
}

export function modSub(a: bigint, b: bigint, mod: bigint): bigint {
  return (a - b + mod) % mod;
}

function modPow(base: bigint, exp: bigint, mod: bigint): bigint {
  let res = 1n;
  base = base % mod;
  while (exp > 0n) {
    if (exp % 2n === 1n) res = (res * base) % mod;
    base = (base * base) % mod;
    exp /= 2n;
  }
  return res;
}

export function pedersenCommit(bid: bigint, randomness: bigint): bigint {
  // Kiểm tra đầu vào
  if (bid >= Q) throw new Error("Bid must be smaller than Q");
  if (randomness >= Q) throw new Error("Randomness must be smaller than Q");

  const term1 = modPow(G, bid, P);

  const term2 = modPow(H, randomness, P);

  return modMul(term1, term2, P);
}
