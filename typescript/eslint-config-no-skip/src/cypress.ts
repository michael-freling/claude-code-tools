import type { Linter } from 'eslint';
import * as mocha from 'eslint-plugin-mocha';

/**
 * ESLint configuration to detect skipped tests in Cypress test files.
 * Applies to *.cy.* files and files in cypress/ directory.
 */
export const cypressConfig: Linter.Config[] = [
  {
    name: 'claude-code-tools/no-skip/cypress',
    files: ['**/*.cy.{js,ts,jsx,tsx}', '**/cypress/**/*.{js,ts,jsx,tsx}'],
    plugins: {
      mocha,
    },
    rules: {
      'mocha/no-skipped-tests': 'error',
      'mocha/no-exclusive-tests': 'error',
    },
  },
];

export default cypressConfig;
