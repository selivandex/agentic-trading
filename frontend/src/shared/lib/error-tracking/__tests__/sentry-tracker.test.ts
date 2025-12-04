/** @format */

import { SentryErrorTracker } from "../sentry-tracker";
import { ErrorLevel } from "../error-tracker";

// Mock Sentry module
const mockCaptureException = jest.fn();
const mockCaptureMessage = jest.fn();
const mockAddBreadcrumb = jest.fn();
const mockConfigureScope = jest.fn();

const mockSentryHub = {
  captureException: mockCaptureException,
  captureMessage: mockCaptureMessage,
  addBreadcrumb: mockAddBreadcrumb,
  configureScope: mockConfigureScope,
};

const mockSentry = {
  getCurrentHub: jest.fn(() => mockSentryHub),
};

// Mock require for Sentry
jest.mock("@sentry/nextjs", () => mockSentry, { virtual: true });

describe("SentryErrorTracker", () => {
  let tracker: SentryErrorTracker;
  let consoleWarnSpy: jest.SpyInstance;
  let consoleErrorSpy: jest.SpyInstance;

  beforeEach(() => {
    jest.clearAllMocks();
    consoleWarnSpy = jest.spyOn(console, "warn").mockImplementation();
    consoleErrorSpy = jest.spyOn(console, "error").mockImplementation();

    tracker = new SentryErrorTracker();
  });

  afterEach(() => {
    consoleWarnSpy.mockRestore();
    consoleErrorSpy.mockRestore();
  });

  describe("captureException", () => {
    it("should capture exception via Sentry", () => {
      const error = new Error("Test error");
      tracker.captureException(error);

      expect(mockCaptureException).toHaveBeenCalledWith(error, undefined);
    });

    it("should include context when provided", () => {
      const error = new Error("Test error");
      const context = {
        tags: { component: "TestComponent" },
        extra: { userId: "123" },
      };

      tracker.captureException(error, context);

      expect(mockCaptureException).toHaveBeenCalledWith(
        error,
        expect.objectContaining({
          tags: { component: "TestComponent" },
          extra: { userId: "123" },
        })
      );
    });

    it("should sanitize sensitive data from tags", () => {
      const error = new Error("Test error");
      const context = {
        tags: {
          component: "TestComponent",
          password: "secret123",
          apiKey: "key123",
        },
      };

      tracker.captureException(error, context);

      expect(mockCaptureException).toHaveBeenCalledWith(
        error,
        expect.objectContaining({
          tags: {
            component: "TestComponent",
            // password and apiKey should be removed
          },
        })
      );

      const capturedContext = mockCaptureException.mock.calls[0][1];
      expect(capturedContext.tags).not.toHaveProperty("password");
      expect(capturedContext.tags).not.toHaveProperty("apiKey");
    });

    it("should sanitize sensitive data from extra", () => {
      const error = new Error("Test error");
      const context = {
        extra: {
          userId: "123",
          userPassword: "secret",
          authToken: "token123",
          data: { secret: "value", normal: "data" },
        },
      };

      tracker.captureException(error, context);

      const capturedContext = mockCaptureException.mock.calls[0][1];
      expect(capturedContext.extra).toHaveProperty("userId");
      expect(capturedContext.extra).not.toHaveProperty("userPassword");
      expect(capturedContext.extra).not.toHaveProperty("authToken");
    });
  });

  describe("captureMessage", () => {
    it("should capture message with correct level", () => {
      tracker.captureMessage("Test message", ErrorLevel.Error);

      expect(mockCaptureMessage).toHaveBeenCalledWith("Test message", {
        level: "error",
      });
    });

    it("should map error levels correctly", () => {
      const levels: Array<[ErrorLevel, string]> = [
        [ErrorLevel.Fatal, "fatal"],
        [ErrorLevel.Error, "error"],
        [ErrorLevel.Warning, "warning"],
        [ErrorLevel.Info, "info"],
        [ErrorLevel.Debug, "debug"],
      ];

      levels.forEach(([ourLevel, sentryLevel]) => {
        mockCaptureMessage.mockClear();
        tracker.captureMessage("Test", ourLevel);

        expect(mockCaptureMessage).toHaveBeenCalledWith(
          "Test",
          expect.objectContaining({ level: sentryLevel })
        );
      });
    });

    it("should include context in message", () => {
      const context = {
        tags: { severity: "high" },
        extra: { details: "test" },
      };

      tracker.captureMessage("Test message", ErrorLevel.Error, context);

      expect(mockCaptureMessage).toHaveBeenCalledWith(
        "Test message",
        expect.objectContaining({
          level: "error",
          tags: { severity: "high" },
          extra: { details: "test" },
        })
      );
    });
  });

  describe("setUser", () => {
    it("should set user via configureScope", () => {
      const user = { id: "123", email: "test@example.com" };
      tracker.setUser(user);

      expect(mockConfigureScope).toHaveBeenCalled();

      // Call the scope callback
      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setUser: jest.fn() };
      scopeCallback(mockScope);

      expect(mockScope.setUser).toHaveBeenCalledWith(
        expect.objectContaining({
          id: "123",
          email: "test@example.com",
        })
      );
    });

    it("should clear user when null is passed", () => {
      tracker.setUser(null);

      expect(mockConfigureScope).toHaveBeenCalled();

      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setUser: jest.fn() };
      scopeCallback(mockScope);

      expect(mockScope.setUser).toHaveBeenCalledWith(null);
    });

    it("should sanitize sensitive user fields", () => {
      const user = {
        id: "123",
        email: "test@example.com",
        password: "secret",
        apiKey: "key123",
        normalField: "value",
      };

      tracker.setUser(user);

      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setUser: jest.fn() };
      scopeCallback(mockScope);

      const sanitizedUser = mockScope.setUser.mock.calls[0][0];
      expect(sanitizedUser).toHaveProperty("id");
      expect(sanitizedUser).toHaveProperty("email");
      expect(sanitizedUser).toHaveProperty("normalField");
      expect(sanitizedUser).not.toHaveProperty("password");
      expect(sanitizedUser).not.toHaveProperty("apiKey");
    });
  });

  describe("addBreadcrumb", () => {
    it("should add breadcrumb via Sentry", () => {
      const breadcrumb = {
        message: "Test breadcrumb",
        category: "test",
      };

      tracker.addBreadcrumb(breadcrumb);

      expect(mockAddBreadcrumb).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "Test breadcrumb",
          category: "test",
        })
      );
    });

    it("should convert timestamp to seconds", () => {
      const breadcrumb = {
        message: "Test",
        timestamp: 1000000,
      };

      tracker.addBreadcrumb(breadcrumb);

      expect(mockAddBreadcrumb).toHaveBeenCalledWith(
        expect.objectContaining({
          timestamp: 1000000, // Already in our format, should be preserved
        })
      );
    });

    it("should generate timestamp if not provided", () => {
      const breadcrumb = {
        message: "Test",
      };

      const before = Date.now() / 1000;
      tracker.addBreadcrumb(breadcrumb);
      const after = Date.now() / 1000;

      const capturedBreadcrumb = mockAddBreadcrumb.mock.calls[0][0];
      expect(capturedBreadcrumb.timestamp).toBeGreaterThanOrEqual(before);
      expect(capturedBreadcrumb.timestamp).toBeLessThanOrEqual(after);
    });
  });

  describe("setContext", () => {
    it("should set context via configureScope", () => {
      const context = { key: "value", nested: { data: "test" } };
      tracker.setContext("custom", context);

      expect(mockConfigureScope).toHaveBeenCalled();

      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setContext: jest.fn() };
      scopeCallback(mockScope);

      expect(mockScope.setContext).toHaveBeenCalledWith("custom", context);
    });

    it("should clear context when null is passed", () => {
      tracker.setContext("custom", null);

      expect(mockConfigureScope).toHaveBeenCalled();

      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setContext: jest.fn() };
      scopeCallback(mockScope);

      expect(mockScope.setContext).toHaveBeenCalledWith("custom", null);
    });
  });

  describe("setTag", () => {
    it("should set tag via configureScope", () => {
      tracker.setTag("environment", "production");

      expect(mockConfigureScope).toHaveBeenCalled();

      const scopeCallback = mockConfigureScope.mock.calls[0][0];
      const mockScope = { setTag: jest.fn() };
      scopeCallback(mockScope);

      expect(mockScope.setTag).toHaveBeenCalledWith("environment", "production");
    });
  });

  describe("error handling", () => {
    it("should handle errors gracefully when Sentry throws", () => {
      mockCaptureException.mockImplementationOnce(() => {
        throw new Error("Sentry error");
      });

      const error = new Error("Test error");

      expect(() => {
        tracker.captureException(error);
      }).not.toThrow();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Failed to capture exception in Sentry:",
        expect.any(Error)
      );
    });

    it("should handle errors when configureScope fails", () => {
      mockConfigureScope.mockImplementationOnce(() => {
        throw new Error("Scope error");
      });

      expect(() => {
        tracker.setUser({ id: "123" });
      }).not.toThrow();

      expect(consoleErrorSpy).toHaveBeenCalledWith(
        "Failed to set user in Sentry:",
        expect.any(Error)
      );
    });
  });
});

