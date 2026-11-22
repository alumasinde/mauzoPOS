import { defineConfig } from 'vitest/config';
import { testDefaults } from '../../vitest.shared.ts';

export default defineConfig({
  test: { ...testDefaults },
});
