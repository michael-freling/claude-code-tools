import type { Linter } from 'eslint';
import { describe, it, expect } from 'vitest';

/**
 * Test that a config exports a non-empty array.
 */
export function testConfigExportsArray(config: Linter.Config[], configName: string): void {
  it('should export an array of configs', () => {
    expect(Array.isArray(config)).toBe(true);
    expect(config.length).toBeGreaterThan(0);
  });
}

interface ConfigStructureTestCase {
  name: string;
  hasLanguageOptions?: boolean;
}

/**
 * Test that a config has the expected structure (name, files, plugins, rules).
 */
export function testConfigStructure(
  config: Linter.Config[],
  expectedConfigName: string,
  options: ConfigStructureTestCase = { name: expectedConfigName }
): void {
  describe('config structure', () => {
    it('should have name property', () => {
      expect(config[0].name).toBe(expectedConfigName);
    });

    it('should have files array', () => {
      expect(Array.isArray(config[0].files)).toBe(true);
      expect(config[0].files!.length).toBeGreaterThan(0);
    });

    it('should have plugins object', () => {
      expect(config[0].plugins).toBeDefined();
      expect(typeof config[0].plugins).toBe('object');
    });

    it('should have rules object', () => {
      expect(config[0].rules).toBeDefined();
      expect(typeof config[0].rules).toBe('object');
    });

    if (options.hasLanguageOptions) {
      it('should have languageOptions with globals', () => {
        expect(config[0].languageOptions).toBeDefined();
        expect(config[0].languageOptions?.globals).toBeDefined();
      });
    }
  });
}

interface FilePatternTestCase {
  pattern: string;
  description: string;
}

/**
 * Test that expected file patterns are present in the config.
 */
export function testFilePatterns(config: Linter.Config[], patterns: FilePatternTestCase[]): void {
  describe('file patterns', () => {
    const configFiles = config[0].files;
    patterns.forEach(({ pattern, description }) => {
      it(`should include pattern for ${description}: ${pattern}`, () => {
        expect(configFiles).toContain(pattern);
      });
    });
  });
}

/**
 * Test that a plugin is configured.
 */
export function testPluginConfigured(config: Linter.Config[], pluginName: string): void {
  describe('plugin configuration', () => {
    it('should have plugin configured', () => {
      expect(config[0].plugins?.[pluginName]).toBeDefined();
    });
  });
}

interface RuleTestCase {
  rule: string;
  level: string;
  description: string;
}

/**
 * Test that rules are set to expected severity.
 */
export function testRulesConfiguration(config: Linter.Config[], rules: RuleTestCase[]): void {
  describe('rules configuration', () => {
    const configRules = config[0].rules;
    rules.forEach(({ rule, level, description }) => {
      it(`${description} (${rule} = ${level})`, () => {
        expect(configRules?.[rule]).toBe(level);
      });
    });
  });
}

interface GlobalTestCase {
  name: string;
  type: string;
}

/**
 * Test that language options globals are configured correctly.
 */
export function testLanguageOptionsGlobals(config: Linter.Config[], globals: GlobalTestCase[]): void {
  describe('language options', () => {
    it('should define all expected globals', () => {
      const configGlobals = config[0].languageOptions?.globals as Record<string, string> | undefined;
      globals.forEach(({ name, type }) => {
        expect(configGlobals?.[name]).toBe(type);
      });
    });
  });
}
