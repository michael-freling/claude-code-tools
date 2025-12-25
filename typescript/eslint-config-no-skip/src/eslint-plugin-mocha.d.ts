declare module 'eslint-plugin-mocha' {
  import type { ESLint, Linter } from 'eslint';

  const plugin: ESLint.Plugin & {
    configs: Record<string, Linter.Config>;
  };

  export = plugin;
}
