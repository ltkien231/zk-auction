import { G, H, P, Q, L, N } from "./constants";
import { randomInt, pedersenCommit } from "./math";

export class Bidder {
  id: number;
  bid: number;
  salt: number;
  commitment: bigint;
  privX: bigint[] = [];
  pubX: bigint[] = [];
  privS: bigint[] = [];
  pubS: bigint[] = [];
  Ti: bigint[] = [];

  constructor(id: number, bid: number) {
    this.id = id;
    this.bid = bid;
    this.salt = randomInt(10);
    this.commitment = pedersenCommit(BigInt(bid), BigInt(this.salt));
    for (let j = 0; j < L; j++) {
      const x = BigInt(randomInt(10));
      const s = BigInt(randomInt(10));
      this.privX.push(x);
      this.pubX.push(G ** x);
      this.privS.push(s);
      this.pubS.push(H ** s);
    }
  }

  computeTi(publicX: bigint[][]) {
    for (let j = 0; j < L; j++) {
      let preProd = BigInt(1);
      let postProd = BigInt(1);
      for (let k = 0; k < this.id; k++) {
        preProd = preProd * publicX[k][j];
      }
      for (let k = this.id + 1; k < N; k++) {
        postProd = postProd * publicX[k][j];
      }
      this.Ti.push(preProd / postProd);
    }
  }
}
