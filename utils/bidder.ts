import { G, H, P, Q, L, N } from "./constants";
import { ranBigint, pedersenCommit, modPow, modMul, modDiv, modSub } from "./math";
import {keccak256, encodePacked} from 'viem'

type XSProof ={
  gPowC1: bigint;
  gPowC2: bigint;
  r1: bigint;
  r2: bigint;
}

export class Bidder {
  id: number;
  bid: number;
  bidBinary: number[] = [];
  salt: bigint;
  commitment: bigint;
  privX: bigint[] = [];
  pubX: bigint[] = [];
  privS: bigint[] = [];
  pubS: bigint[] = [];
  xsProof: XSProof[] = [];
  bitZeroCommitments: bigint[] = [];
  bitOneCommitments: bigint[] = [];
  publicXs: readonly (readonly bigint[])[] = [];
  isLost: boolean = false;
  c1: bigint = 0n;
  c2: bigint = 0n;
  c: bigint = 0n;

  constructor(id: number, bid: number) {
    this.id = id;

    this.bid = bid;
    this.bidBinary = numberToBinaryArray(bid, L);

    this.salt = ranBigint(Q);
    this.c = ranBigint(Q);
    this.c1 = ranBigint(Q);
    this.c2 = ranBigint(Q);
    
    this.commitment = pedersenCommit(BigInt(bid), BigInt(this.salt));

    for (let j = 0; j < L; j++) {
      const x = ranBigint(Q);
      this.privX.push(x);
      this.pubX.push(modPow(G, x, P));

      const s = ranBigint(Q);
      this.privS.push(s);
      this.pubS.push(modPow(H, s, P));

      // ZKP
      const xsProof: XSProof = {
        gPowC1: modPow(G, this.c1, P),
        gPowC2: modPow(G, this.c2, P),
        r1: 0n,
        r2: 0n,
      };
      
      const packed = encodePacked(['uint256', 'uint256', 'uint256', 'uint256', 'uint256', 'uint256', 'uint256'], [G, xsProof.gPowC1, xsProof.gPowC2, modPow(G, x, P), modPow(G, s, P), BigInt(this.id), BigInt(j)]);
      const hash = BigInt(keccak256(packed))

      xsProof.r1 = modSub(this.c1, modMul(x, hash, Q), Q); 
      xsProof.r2 = modSub(this.c2, modMul(s, hash, Q), Q); 
      this.xsProof.push(xsProof);
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

