# @michael-freling/eslint-config-no-skip

ESLint shareable configuration to detect and error on skipped tests in Jest and Cypress test suites.

This package provides ESLint configurations that prevent accidentally committing skipped tests (`.skip`) or exclusive tests (`.only`) in your test files. It ensures your CI/CD pipeline runs all tests, not just a subset.

## Installation

This package is published to GitHub Packages. You need to configure npm/pnpm to use GitHub Packages for the `@michael-freling` scope.

### Step 1: Configure GitHub Packages registry

Add this to your project's `.npmrc` file (create one if it doesn't exist):

```
@michael-freling:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=${GITHUB_TOKEN}
```

Or run:

```bash
echo "@michael-freling:registry=https://npm.pkg.github.com" >> .npmrc
```

### Step 2: Authenticate with GitHub Packages

Set your GitHub token (needs `read:packages` scope):

```bash
# Using GitHub CLI
export GITHUB_TOKEN=$(gh auth token)

# Or set it directly
export GITHUB_TOKEN=your_github_token
```

### Step 3: Install the package

```bash
pnpm add -D @michael-freling/eslint-config-no-skip
```

Also install peer dependencies:

```bash
pnpm add -D eslint eslint-plugin-jest eslint-plugin-mocha
```

## Requirements

- ESLint 9.0.0 or higher (for flat config)
- Node.js 18 or higher

## Usage

### ESLint 9 Flat Config (Recommended)

This package is designed for ESLint 9's flat config format (`eslint.config.js`).

#### Combined (Jest + Cypress)

Import the default configuration to check both Jest and Cypress test files:

```javascript
// eslint.config.js
import noSkipConfig from '@michael-freling/eslint-config-no-skip';

export default [
  ...noSkipConfig,
  // your other configs
];
```

This configuration will:
- Apply Jest rules to `**/*.test.{js,ts,jsx,tsx}` and `**/*.spec.{js,ts,jsx,tsx}` files
- Apply Cypress rules to `**/*.cy.{js,ts,jsx,tsx}` and `**/cypress/**/*.{js,ts,jsx,tsx}` files

#### Jest Only

Import only the Jest configuration:

```javascript
// eslint.config.js
import jestConfig from '@michael-freling/eslint-config-no-skip/jest';

export default [
  ...jestConfig,
  // your other configs
];
```

#### Cypress Only

Import only the Cypress configuration:

```javascript
// eslint.config.js
import cypressConfig from '@michael-freling/eslint-config-no-skip/cypress';

export default [
  ...cypressConfig,
  // your other configs
];
```

### Legacy .eslintrc.json Configuration

If you're using ESLint 8 or the legacy `.eslintrc.json` format, you cannot use this package directly. Instead, configure the underlying plugins manually:

```json
{
  "plugins": ["jest", "mocha"],
  "overrides": [
    {
      "files": ["**/*.test.{js,ts,jsx,tsx}", "**/*.spec.{js,ts,jsx,tsx}"],
      "env": { "jest": true },
      "rules": {
        "jest/no-disabled-tests": "error"
      }
    },
    {
      "files": ["**/*.cy.{js,ts,jsx,tsx}", "**/cypress/**/*.{js,ts,jsx,tsx}"],
      "rules": {
        "mocha/no-skipped-tests": "error",
        "mocha/no-exclusive-tests": "error"
      }
    }
  ]
}
```

Install the plugins:

```bash
pnpm add -D eslint-plugin-jest eslint-plugin-mocha
```

## Detected Patterns

### Jest Tests

| Pattern | Rule | Description |
| --- | --- | --- |
| `describe.skip()` | `jest/no-disabled-tests` | Skipped test suite |
| `it.skip()` | `jest/no-disabled-tests` | Skipped test case |
| `test.skip()` | `jest/no-disabled-tests` | Skipped test case |
| `xdescribe()` | `jest/no-disabled-tests` | Skipped test suite (alternative syntax) |
| `xit()` | `jest/no-disabled-tests` | Skipped test case (alternative syntax) |
| `xtest()` | `jest/no-disabled-tests` | Skipped test case (alternative syntax) |

### Cypress Tests

| Pattern | Rule | Description |
| --- | --- | --- |
| `describe.skip()` | `mocha/no-skipped-tests` | Skipped test suite |
| `it.skip()` | `mocha/no-skipped-tests` | Skipped test case |
| `context.skip()` | `mocha/no-skipped-tests` | Skipped test context |
| `describe.only()` | `mocha/no-exclusive-tests` | Exclusive test suite |
| `it.only()` | `mocha/no-exclusive-tests` | Exclusive test case |
| `context.only()` | `mocha/no-exclusive-tests` | Exclusive test context |

## Troubleshooting

### Authentication errors with GitHub Packages

If you see `401 Unauthorized` or `403 Forbidden` errors:

1. Ensure your GitHub token has `read:packages` scope
2. Verify your `.npmrc` is configured correctly
3. Check that `GITHUB_TOKEN` environment variable is set

### Config not being applied

Ensure your test files match the configured patterns:
- Jest: `**/*.test.{js,ts,jsx,tsx}` or `**/*.spec.{js,ts,jsx,tsx}`
- Cypress: `**/*.cy.{js,ts,jsx,tsx}` or `**/cypress/**/*.{js,ts,jsx,tsx}`

If your test files use different naming conventions, you can customize the patterns by creating your own config based on the exported configurations.

## License

MIT
