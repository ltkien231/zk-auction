import { G_POINT, H_POINT, G_ZERO, N, L } from "./constants";
import {
  G1Point,
  G1PointViem,
  randomScalar,
  scalarMul,
  pointAdd,
  pointSub,
  pedersenCommit,
  pointToViem,
  viemToPoint,
  intToBitsMSB,
} from "./math";

export class Bidder {
  id: number;
  bid: number;
  /** MSB-first binary representation (bidBinary[0] = MSB). */
  bidBinary: number[];
  salt: bigint;
  isLost = false;

  private _commitment: G1Point;
  private _privX: bigint[]   = [];
  private _pubX:  G1Point[]  = [];
  private _privS: bigint[]   = [];
  private _pubS:  G1Point[]  = [];
  private _bitZeroCommits: G1Point[] = [];
  private _bitOneCommits:  G1Point[] = [];

  constructor(id: number, bid: number) {
    this.id        = id;
    this.bid       = bid;
    this.bidBinary = intToBitsMSB(bid, L);
    this.salt      = randomScalar();
    this._commitment = pedersenCommit(BigInt(bid), this.salt);

    for (let j = 0; j < L; j++) {
      const x = randomScalar();
      this._privX.push(x);
      this._pubX.push(scalarMul(G_POINT, x));

      const s = randomScalar();
      this._privS.push(s);
      this._pubS.push(scalarMul(H_POINT, s));
    }
  }

  // ─── Viem-ready getters (for contract calls) ───────────────────────────────

  get commitment(): G1PointViem { return pointToViem(this._commitment); }
  get pubX(): G1PointViem[]     { return this._pubX.map(pointToViem); }
  get pubS(): G1PointViem[]     { return this._pubS.map(pointToViem); }

  get bitZeroCommitments(): G1PointViem[] { return this._bitZeroCommits.map(pointToViem); }
  get bitOneCommitments():  G1PointViem[] { return this._bitOneCommits.map(pointToViem); }

  // ─── AV-net round-2 tally key computation ────────────────────────────────

  /**
   * Compute per-bit cryptograms after all bidders have joined.
   * @param allPubXs  2D array from contract.getPublicXs():
   *                  allPubXs[i][j] = bidder i's X key for bit position j.
   */
  computeBitCommitments(allPubXs: readonly (readonly G1PointViem[])[]) {
    this._bitZeroCommits = [];
    this._bitOneCommits  = [];

    const points = allPubXs.map((row) => row.map(viemToPoint));

    for (let j = 0; j < L; j++) {
      // T_i = (∑_{k<i} X_k[j]) - (∑_{k>i} X_k[j])
      let pre:  G1Point = G_ZERO;
      let post: G1Point = G_ZERO;

      for (let k = 0; k < this.id; k++)      pre  = pointAdd(pre,  points[k][j]);
      for (let k = this.id + 1; k < N; k++)  post = pointAdd(post, points[k][j]);

      const Ti = pointSub(pre, post);

      // bit=0 (has 0 at this position): s_j * T_i
      // bit=1 (has 1 at this position, or already lost): x_j * T_i
      this._bitZeroCommits.push(scalarMul(Ti, this._privS[j]));
      this._bitOneCommits.push( scalarMul(Ti, this._privX[j]));
    }
  }
}
