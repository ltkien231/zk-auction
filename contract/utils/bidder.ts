import { G, H, P, Q, L, N } from "./constants";
import { randomInt, pedersenCommit, modPow, modMul, modDiv } from "./math";

export class Bidder {
  id: number;
  bid: number;
  bidBinary: number[] = [];
  salt: number;
  commitment: bigint;
  privX: bigint[] = [];
  pubX: bigint[] = [];
  privS: bigint[] = [];
  pubS: bigint[] = [];
  bitZeroCommitments: bigint[] = [];
  bitOneCommitments: bigint[] = [];
  publicXs: readonly (readonly bigint[])[] = [];
  isLost: boolean = false;

  constructor(id: number, bid: number) {
    this.id = id;

    this.bid = bid;
    this.bidBinary = numberToBinaryArray(bid, L);

    this.salt = randomInt(Number(Q));
    this.commitment = pedersenCommit(BigInt(bid), BigInt(this.salt));

    for (let j = 0; j < L; j++) {
      const x = BigInt(randomInt(Number(Q)));
      this.privX.push(x);
      this.pubX.push(modPow(G, x, P));

      const s = BigInt(randomInt(Number(Q)));
      this.privS.push(s);
      this.pubS.push(modPow(H, s, P));
    }
  }

  computeBitCommitments(pubXs: readonly (readonly bigint[])[]) {
    for (let j = 0; j < L; j++) {
      let preProd = 1n;
      let postProd = 1n;

      for (let k = 0; k < this.id; k++) {
        preProd = modMul(preProd, pubXs[k][j], P);
      }
      for (let k = this.id + 1; k < N; k++) {
        postProd = modMul(postProd, pubXs[k][j], P);
      }

      const Ti = modDiv(preProd, postProd, P);
      this.bitZeroCommitments.push(modPow(Ti, this.privS[j], P));
      this.bitOneCommitments.push(modPow(Ti, this.privX[j], P));
    }
  }
}

function numberToBinaryArray(num: number, length: number): number[] {
  return num.toString(2).padStart(length, "0").split("").map(Number);
}
