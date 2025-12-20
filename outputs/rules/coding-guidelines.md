# Coding Guidelines

- Read a guideline file **.claude/docs/guideline.md**
- Simplicity is the most important thing.
- When installing applications, libraries, or tools, always check and use the most latest and stable version with compatibility with existing systems.
- DO NOT IGNORE pre-commits errors and fix them properly.
- DO NOT USE general terms like shared, common, utils or info for naming variables, functions, classes, tables, and so on.

## Coding

- Write the code with minimal comments — only high-level explanations of purpose, architecture, or non-obvious decisions. No line-by-line comments
- Delete deadcodes.
- Reuse the existing codes as much as possible, and avoid duplicating codes.
- Delete assignments of the default or zero values.
- Every error must be checked or returned.
- **Prefer to continue or return early** than nesting code
   - "if is bad, else is worse"

## Testing

- DO NOT SKIP test failures. Fix those test cases to pass
- **Use table-driven testing**
    - Split happy and error test sets if complicated
    - Reduces code duplication and improves maintainability
- **Define test inputs as test case fields**, not as function arguments
- Write code that operates identically across dev, test, and production. Avoid environment-specific logic in core logic. Use configuration or dependency injection instead of branching code. Use test doubles externally, not via conditionals in production code. No hacks, no assumptions, no global state. Always default to production-safe behavior.
    - Use dependency injection in order to make testing easier

## Security

- Set proper owners and permissions instead of setting 777 to files or directories
