/** @format */

import { get } from "../utils";

describe("CRUD Utils", () => {
  describe("get", () => {
    it("should get nested value by path", () => {
      const obj = {
        data: {
          user: {
            name: "John",
            age: 30,
          },
        },
      };

      expect(get(obj, "data.user.name")).toBe("John");
      expect(get(obj, "data.user.age")).toBe(30);
    });

    it("should return default value for non-existent path", () => {
      const obj = { data: { user: { name: "John" } } };

      expect(get(obj, "data.user.email", "default")).toBe("default");
      expect(get(obj, "missing.path", null)).toBe(null);
    });

    it("should handle null/undefined gracefully", () => {
      expect(get(null, "path", "default")).toBe("default");
      expect(get(undefined, "path", "default")).toBe("default");
    });

    it("should handle arrays", () => {
      const obj = {
        items: [
          { id: 1, name: "Item 1" },
          { id: 2, name: "Item 2" },
        ],
      };

      expect(get(obj, "items")).toEqual(obj.items);
    });
  });
});
