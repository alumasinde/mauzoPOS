// Shared Vitest settings: single worker thread, generous startup timeout
// (Windows dev machines can be slow to spawn workers).
export const testDefaults = {
  pool: 'threads' as const,
  maxWorkers: 1,
  teardownTimeout: 30_000,
  testTimeout: 15_000,
};
