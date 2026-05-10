import { describe, it, expect } from 'vitest';
import noSkipConfig, { jestConfig, cypressConfig } from '../index.js';

describe('noSkipConfig', () => {
  it('should export an array of configs', () => {
    expect(Array.isArray(noSkipConfig)).toBe(true);
    expect(noSkipConfig.length).toBeGreaterThan(0);
  });

  describe('combined config includes all sub-configs', () => {
    const testCases = [
      {
        name: 'should include Jest config',
        expectedCount: 1,
        configSource: jestConfig,
      },
      {
        name: 'should include Cypress config',
        expectedCount: 1,
        configSource: cypressConfig,
      },
    ];

    testCases.forEach(({ name, expectedCount, configSource }) => {
      it(name, () => {
        expect(configSource.length).toBe(expectedCount);
        const configNames = noSkipConfig.map((c) => c.name);
        configSource.forEach((config) => {
          expect(configNames).toContain(config.name);
        });
      });
    });
  });

  describe('config ordering and structure', () => {
    it('should have both Jest and Cypress configs', () => {
      const configNames = noSkipConfig.map((c) => c.name);
      expect(configNames).toContain('claude-code-tools/no-skip/jest');
      expect(configNames).toContain('claude-code-tools/no-skip/cypress');
    });

    it('should have correct total number of configs', () => {
      expect(noSkipConfig.length).toBe(jestConfig.length + cypressConfig.length);
    });
  });

  describe('named exports', () => {
    const testCases = [
      {
        name: 'jestConfig',
        exportValue: jestConfig,
        expectedLength: 1,
        expectedName: 'claude-code-tools/no-skip/jest',
      },
      {
        name: 'cypressConfig',
        exportValue: cypressConfig,
        expectedLength: 1,
        expectedName: 'claude-code-tools/no-skip/cypress',
      },
    ];

    testCases.forEach(({ name, exportValue, expectedLength, expectedName }) => {
      it(`should export ${name} as array with ${expectedLength} config(s)`, () => {
        expect(Array.isArray(exportValue)).toBe(true);
        expect(exportValue.length).toBe(expectedLength);
        expect(exportValue[0].name).toBe(expectedName);
      });
    });
  });

  describe('config completeness', () => {
    it('should have all configs with required properties', () => {
      noSkipConfig.forEach((config) => {
        expect(config.name).toBeDefined();
        expect(typeof config.name).toBe('string');
        expect(config.files).toBeDefined();
        expect(Array.isArray(config.files)).toBe(true);
        expect(config.plugins).toBeDefined();
        expect(typeof config.plugins).toBe('object');
        expect(config.rules).toBeDefined();
        expect(typeof config.rules).toBe('object');
      });
    });

    const expectedConfigs = [
      {
        name: 'claude-code-tools/no-skip/jest',
        expectedRules: ['jest/no-disabled-tests'],
      },
      {
        name: 'claude-code-tools/no-skip/cypress',
        expectedRules: ['mocha/no-skipped-tests', 'mocha/no-exclusive-tests'],
      },
    ];

    expectedConfigs.forEach(({ name, expectedRules }) => {
      it(`should have all expected rules for ${name}`, () => {
        const config = noSkipConfig.find((c) => c.name === name);
        expect(config).toBeDefined();
        expectedRules.forEach((rule) => {
          expect(config?.rules?.[rule]).toBe('error');
        });
      });
    });
  });
});
