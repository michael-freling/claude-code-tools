import { describe } from 'vitest';
import { cypressConfig } from '../cypress.js';
import {
  testConfigExportsArray,
  testConfigStructure,
  testFilePatterns,
  testPluginConfigured,
  testRulesConfiguration,
} from './testHelpers.js';

describe('cypressConfig', () => {
  testConfigExportsArray(cypressConfig, 'cypressConfig');

  testConfigStructure(cypressConfig, 'claude-code-tools/no-skip/cypress');

  testFilePatterns(cypressConfig, [
    { pattern: '**/*.cy.{js,ts,jsx,tsx}', description: 'Cypress test files' },
    { pattern: '**/cypress/**/*.{js,ts,jsx,tsx}', description: 'Cypress directory files' },
  ]);

  testPluginConfigured(cypressConfig, 'mocha');

  testRulesConfiguration(cypressConfig, [
    {
      rule: 'mocha/no-skipped-tests',
      level: 'error',
      description: 'should error on skipped tests',
    },
    {
      rule: 'mocha/no-exclusive-tests',
      level: 'error',
      description: 'should error on exclusive tests',
    },
  ]);
});
