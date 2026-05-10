declare module 'eslint-plugin-jest' {
  import type { ESLint, Linter } from 'eslint';

  const plugin: ESLint.Plugin & {
    configs: Record<string, Linter.Config>;
  };

  export = plugin;
}
