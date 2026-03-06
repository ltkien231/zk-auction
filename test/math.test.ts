import { expect } from "chai";
import { intToBits, bitsToInt, intToBitsMSB, pedersenCommit, pointToViem, viemToPoint, G_POINT, H_POINT, randomScalar } from "../utils";

describe("Math Utils", function () {
  describe("intToBits and bitsToInt (LSB-first)", function () {
    it("should convert integer to bits and back", function () {
      const n = 13; // binary: 1101
      const bits = intToBits(n, 4);
      expect(bits).to.deep.equal([1, 0, 1, 1]); // LSB first
      expect(bitsToInt(bits)).to.equal(n);
    });
  });

  describe("intToBitsMSB", function () {
    it("should produce MSB-first bit array", function () {
      expect(intToBitsMSB(13, 4)).to.deep.equal([1, 1, 0, 1]);
      expect(intToBitsMSB(8, 4)).to.deep.equal([1, 0, 0, 0]);
    });
  });

  describe("G1 point encode/decode roundtrip", function () {
    it("should encode and decode G_POINT", function () {
      const viem = pointToViem(G_POINT);
      const recovered = viemToPoint(viem);
      expect(recovered.equals(G_POINT)).to.be.true;
    });

    it("should encode and decode H_POINT", function () {
      const viem = pointToViem(H_POINT);
      const recovered = viemToPoint(viem);
      expect(recovered.equals(H_POINT)).to.be.true;
    });

    it("should encode and decode a random scalar multiplication", function () {
      const s = randomScalar();
      const p = G_POINT.multiply(s);
      const recovered = viemToPoint(pointToViem(p));
      expect(recovered.equals(p)).to.be.true;
    });
  });

  describe("Pedersen commitment", function () {
    it("should be deterministic given the same inputs", function () {
      const bid = 42n;
      const r   = 123456789n;
      const c1  = pedersenCommit(bid, r);
      const c2  = pedersenCommit(bid, r);
      expect(c1.equals(c2)).to.be.true;
    });

    it("different bids produce different commitments", function () {
      const r  = randomScalar();
      const c1 = pedersenCommit(100n, r);
      const c2 = pedersenCommit(101n, r);
      expect(c1.equals(c2)).to.be.false;
    });
  });
});
