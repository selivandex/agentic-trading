/** @format */

import { ConsoleErrorTracker } from "../console-tracker";
import { ErrorLevel } from "../error-tracker";

// Mock logger
jest.mock("@/shared/lib/logger", () => ({
  logger: {
    error: jest.fn(),
    warn: jest.fn(),
    info: jest.fn(),
    debug: jest.fn(),
  },
}));

// Import logger after mock
import { logger } from "@/shared/lib/logger";

describe("ConsoleErrorTracker", () => {
  let tracker: ConsoleErrorTracker;

  beforeEach(() => {
    tracker = new ConsoleErrorTracker();
    jest.clearAllMocks();
  });

  describe("captureException", () => {
    it("should log error with details", () => {
      const error = new Error("Test error");
      tracker.captureException(error);

      expect(logger.error).toHaveBeenCalledWith(
        "Error captured:",
        expect.objectContaining({
          error: {
            name: "Error",
            message: "Test error",
            stack: expect.any(String),
          },
        })
      );
    });

    it("should include context in log", () => {
      const error = new Error("Test error");
      const context = {
        tags: { component: "TestComponent" },
        extra: { userId: "123" },
      };

      tracker.captureException(error, context);

      expect(logger.error).toHaveBeenCalledWith(
        "Error captured:",
        expect.objectContaining({
          error: expect.any(Object),
          tags: { component: "TestComponent" },
          extra: { userId: "123" },
        })
      );
    });

    it("should include user context if set", () => {
      const error = new Error("Test error");
      tracker.setUser({ id: "user123", email: "test@example.com" });
      tracker.captureException(error);

      expect(logger.error).toHaveBeenCalledWith(
        "Error captured:",
        expect.objectContaining({
          user: { id: "user123", email: "test@example.com" },
        })
      );
    });

    it("should log breadcrumbs if any exist", () => {
      tracker.addBreadcrumb({ message: "User clicked button" });
      tracker.addBreadcrumb({ message: "Navigation occurred" });

      const error = new Error("Test error");
      tracker.captureException(error);

      expect(logger.debug).toHaveBeenCalledWith(
        "Breadcrumbs:",
        expect.arrayContaining([
          expect.objectContaining({ message: "User clicked button" }),
          expect.objectContaining({ message: "Navigation occurred" }),
        ])
      );
    });
  });

  describe("captureMessage", () => {
    it("should log error level messages", () => {
      tracker.captureMessage("Error message", ErrorLevel.Error);

      expect(logger.error).toHaveBeenCalledWith("Error message", {});
    });

    it("should log fatal level messages as error", () => {
      tracker.captureMessage("Fatal message", ErrorLevel.Fatal);

      expect(logger.error).toHaveBeenCalledWith("Fatal message", {});
    });

    it("should log warning level messages", () => {
      tracker.captureMessage("Warning message", ErrorLevel.Warning);

      expect(logger.warn).toHaveBeenCalledWith("Warning message", {});
    });

    it("should log info level messages", () => {
      tracker.captureMessage("Info message", ErrorLevel.Info);

      expect(logger.info).toHaveBeenCalledWith("Info message", {});
    });

    it("should log debug level messages", () => {
      tracker.captureMessage("Debug message", ErrorLevel.Debug);

      expect(logger.debug).toHaveBeenCalledWith("Debug message", {});
    });

    it("should include context in message log", () => {
      const context = {
        tags: { severity: "high" },
        extra: { details: "test" },
      };

      tracker.captureMessage("Test message", ErrorLevel.Error, context);

      expect(logger.error).toHaveBeenCalledWith(
        "Test message",
        expect.objectContaining({
          tags: { severity: "high" },
          extra: { details: "test" },
        })
      );
    });
  });

  describe("setUser", () => {
    it("should log when user context is set", () => {
      const user = { id: "123", email: "test@example.com" };
      tracker.setUser(user);

      expect(logger.debug).toHaveBeenCalledWith("User context set:", user);
    });

    it("should log when user context is cleared", () => {
      tracker.setUser(null);

      expect(logger.debug).toHaveBeenCalledWith("User context cleared");
    });
  });

  describe("addBreadcrumb", () => {
    it("should log breadcrumb with timestamp", () => {
      const breadcrumb = {
        message: "Test breadcrumb",
        category: "test",
      };

      tracker.addBreadcrumb(breadcrumb);

      expect(logger.debug).toHaveBeenCalledWith(
        "Breadcrumb added:",
        expect.objectContaining({
          message: "Test breadcrumb",
          category: "test",
          timestamp: expect.any(Number),
        })
      );
    });

    it("should use provided timestamp if given", () => {
      const timestamp = 1234567890;
      tracker.addBreadcrumb({
        message: "Test",
        timestamp,
      });

      expect(logger.debug).toHaveBeenCalledWith(
        "Breadcrumb added:",
        expect.objectContaining({
          timestamp,
        })
      );
    });

    it("should limit breadcrumbs to 100", () => {
      // Add 150 breadcrumbs
      for (let i = 0; i < 150; i++) {
        tracker.addBreadcrumb({ message: `Breadcrumb ${i}` });
      }

      // Capture error to log breadcrumbs
      tracker.captureException(new Error("Test"));

      // Check that breadcrumbs array has max 100 items
      const breadcrumbsCall = (logger.debug as unknown as jest.Mock).mock.calls.find(
        (call) => call[0] === "Breadcrumbs:"
      );

      expect(breadcrumbsCall).toBeDefined();
      expect(breadcrumbsCall[1]).toHaveLength(100);
    });
  });

  describe("setContext", () => {
    it("should log when context is set", () => {
      const context = { key: "value" };
      tracker.setContext("test", context);

      expect(logger.debug).toHaveBeenCalledWith(
        'Context "test" set:',
        context
      );
    });

    it("should log when context is cleared", () => {
      tracker.setContext("test", null);

      expect(logger.debug).toHaveBeenCalledWith('Context "test" cleared');
    });
  });

  describe("setTag", () => {
    it("should log when tag is set", () => {
      tracker.setTag("environment", "production");

      expect(logger.debug).toHaveBeenCalledWith(
        'Tag "environment" set:',
        "production"
      );
    });

    it("should include tags in error logs", () => {
      tracker.setTag("component", "TestComponent");
      tracker.setTag("version", "1.0.0");

      tracker.captureException(new Error("Test"));

      expect(logger.error).toHaveBeenCalledWith(
        "Error captured:",
        expect.objectContaining({
          tags: {
            component: "TestComponent",
            version: "1.0.0",
          },
        })
      );
    });
  });
});
