/** @format */

// Learn more: https://github.com/testing-library/jest-dom
require("@testing-library/jest-dom");

// Setup jest-axe for accessibility testing
const { toHaveNoViolations } = require("jest-axe");
expect.extend(toHaveNoViolations);

// Add fetch polyfill for Apollo Client
global.fetch = jest.fn(() =>
  Promise.resolve({
    json: () => Promise.resolve({}),
  })
);

// Mock uuid to avoid ESM issues
jest.mock("uuid", () => ({
  v4: () => "test-uuid-1234",
}));

// Mock EmptyState to avoid ESM issues with hast-util-to-jsx-runtime
const React = require('react');

jest.mock('@/shared/application', () => {
  const MockEmptyState = ({ children }) =>
    React.createElement('div', { 'data-testid': 'empty-state' }, children);

  MockEmptyState.Header = ({ children }) =>
    React.createElement('div', { 'data-testid': 'empty-state-header' }, children);

  MockEmptyState.Content = ({ children }) =>
    React.createElement('div', { 'data-testid': 'empty-state-content' }, children);

  MockEmptyState.Title = ({ children }) =>
    React.createElement('div', { 'data-testid': 'empty-state-title' }, children);

  MockEmptyState.Description = ({ children }) =>
    React.createElement('div', { 'data-testid': 'empty-state-description' }, children);

  MockEmptyState.FeaturedIcon = () =>
    React.createElement('div', { 'data-testid': 'empty-state-icon' });

  return {
    EmptyState: MockEmptyState,
  };
});
