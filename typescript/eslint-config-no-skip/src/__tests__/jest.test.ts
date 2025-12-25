import { describe } from 'vitest';
import { jestConfig } from '../jest.js';
import {
  testConfigExportsArray,
  testConfigStructure,
  testFilePatterns,
  testPluginConfigured,
  testRulesConfiguration,
  testLanguageOptionsGlobals,
} from './testHelpers.js';

describe('jestConfig', () => {
  testConfigExportsArray(jestConfig, 'jestConfig');

  testConfigStructure(jestConfig, 'claude-code-tools/no-skip/jest', {
    name: 'claude-code-tools/no-skip/jest',
    hasLanguageOptions: true,
  });

  testFilePatterns(jestConfig, [
    { pattern: '**/*.test.{js,ts,jsx,tsx}', description: 'test files' },
    { pattern: '**/*.spec.{js,ts,jsx,tsx}', description: 'spec files' },
  ]);

  testPluginConfigured(jestConfig, 'jest');

  testLanguageOptionsGlobals(jestConfig, [
    { name: 'describe', type: 'readonly' },
    { name: 'it', type: 'readonly' },
    { name: 'test', type: 'readonly' },
    { name: 'expect', type: 'readonly' },
    { name: 'beforeAll', type: 'readonly' },
    { name: 'afterAll', type: 'readonly' },
    { name: 'beforeEach', type: 'readonly' },
    { name: 'afterEach', type: 'readonly' },
    { name: 'jest', type: 'readonly' },
  ]);

  testRulesConfiguration(jestConfig, [
    {
      rule: 'jest/no-disabled-tests',
      level: 'error',
      description: 'should error on disabled tests',
    },
  ]);
});
