import "@testing-library/jest-dom/vitest";

// jsdom provides window.confirm as a stub that returns false; components such
// as CancelRunButton rely on it, so make it default to confirming in tests.
if (typeof window !== "undefined" && !window.confirm) {
  window.confirm = () => true;
}
