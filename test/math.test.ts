import { expect } from "chai";
import { intToBits, bitsToInt, modAdd, modSub, modMul, modDiv, modInv, modPow, G, P } from "../utils";

describe("Math Utils", function () {
  describe("intToBits and bitsToInt", function () {
    it("should convert integer to bits and back", function () {
      const n = 13; // binary: 1101
      const width = 4;
      const bits = intToBits(n, width);
      expect(bits).to.deep.equal([1, 0, 1, 1]);
      const reconstructed = bitsToInt(bits);
      expect(reconstructed).to.equal(n);
    });
  });

  describe("Modular Arithmetic", function () {
    const mod = 17n;

    it("should perform modular addition", function () {
      const result = modAdd(15n, 10n, mod);
      expect(result).to.equal(8n);
    });

    it("should perform modular subtraction", function () {
      const result = modSub(5n, 10n, mod);
      expect(result).to.equal(12n);
    });

    it("should perform modular multiplication", function () {
      const result = modMul(4n, 5n, mod);
      expect(result).to.equal(3n);
    });

    it("should compute modular inverse", function () {
      const result = modInv(3n, mod);
      expect(result).to.equal(6n);
    });

    it("should perform modular division", function () {
      const result = modDiv(4n, 3n, mod);
      expect(result).to.equal(7n);
    });

    it("should perform modular exponentiation", function () {
      const base = 2n;
      const exp = 5n;
      const result = modPow(base, exp, mod);
      expect(result).to.equal(15n);

      expect(modPow(G, 113n, P)).to.equal(435n);
    });
  });
});
