import type { Linter } from 'eslint';
import * as jest from 'eslint-plugin-jest';

/**
 * ESLint configuration to detect skipped tests in Jest test files.
 * Applies to *.test.* and *.spec.* files.
 */
export const jestConfig: Linter.Config[] = [
  {
    name: 'claude-code-tools/no-skip/jest',
    files: ['**/*.test.{js,ts,jsx,tsx}', '**/*.spec.{js,ts,jsx,tsx}'],
    plugins: {
      jest,
    },
    languageOptions: {
      globals: {
        describe: 'readonly',
        it: 'readonly',
        test: 'readonly',
        expect: 'readonly',
        beforeAll: 'readonly',
        afterAll: 'readonly',
        beforeEach: 'readonly',
        afterEach: 'readonly',
        jest: 'readonly',
      },
    },
    rules: {
      'jest/no-disabled-tests': 'error',
    },
  },
];

export default jestConfig;
