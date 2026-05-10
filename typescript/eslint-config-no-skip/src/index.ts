import type { Linter } from 'eslint';
import { jestConfig } from './jest.js';
import { cypressConfig } from './cypress.js';

/**
 * Combined ESLint configuration to detect skipped tests in both Jest and Cypress test files.
 *
 * @example
 * import noSkipConfig from '@claude-code-tools/eslint-config-no-skip';
 * export default [...noSkipConfig];
 */
export const noSkipConfig: Linter.Config[] = [
  ...jestConfig,
  ...cypressConfig,
];

export { jestConfig } from './jest.js';
export { cypressConfig } from './cypress.js';

export default noSkipConfig;
