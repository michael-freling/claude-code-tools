---
paths: **/*.ts, **/*.tsx
---
# TypeScript Coding Guidelines

- Implement UI components based on a mobile-first approach
- Do not output SVG, base64, XML, or any embedded asset data. Use placeholder components or import statements only
- Only use useMemo when there is a real performance need—specifically when a computation is expensive or when memoization prevents unnecessary child re-renders. If useMemo isn't justified, do not add it. When you do use it, add a brief explanation of why it's needed.
