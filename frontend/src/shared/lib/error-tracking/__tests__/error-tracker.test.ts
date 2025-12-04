/** @format */

import { NoopErrorTracker, ErrorLevel } from "../error-tracker";

describe("NoopErrorTracker", () => {
  let tracker: NoopErrorTracker;

  beforeEach(() => {
    tracker = new NoopErrorTracker();
  });

  it("should not throw when capturing exception", () => {
    const error = new Error("Test error");

    expect(() => {
      tracker.captureException(error);
    }).not.toThrow();

    expect(() => {
      tracker.captureException(error, {
        tags: { test: "tag" },
        extra: { data: "value" },
      });
    }).not.toThrow();
  });

  it("should not throw when capturing message", () => {
    expect(() => {
      tracker.captureMessage("Test message", ErrorLevel.Error);
    }).not.toThrow();

    expect(() => {
      tracker.captureMessage("Test message", ErrorLevel.Warning, {
        tags: { test: "tag" },
      });
    }).not.toThrow();
  });

  it("should not throw when setting user", () => {
    expect(() => {
      tracker.setUser({ id: "123", email: "test@example.com" });
    }).not.toThrow();

    expect(() => {
      tracker.setUser(null);
    }).not.toThrow();
  });

  it("should not throw when adding breadcrumb", () => {
    expect(() => {
      tracker.addBreadcrumb({
        message: "Test breadcrumb",
        category: "test",
      });
    }).not.toThrow();
  });

  it("should not throw when setting context", () => {
    expect(() => {
      tracker.setContext("test", { key: "value" });
    }).not.toThrow();

    expect(() => {
      tracker.setContext("test", null);
    }).not.toThrow();
  });

  it("should not throw when setting tag", () => {
    expect(() => {
      tracker.setTag("environment", "test");
    }).not.toThrow();
  });
});

