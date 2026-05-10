import type { Linter } from 'eslint';
import noSkipConfig from './src/index.js';

const config: Linter.Config[] = [
  ...noSkipConfig,
];

export default config;
