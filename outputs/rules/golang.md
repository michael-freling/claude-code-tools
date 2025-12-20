---
paths: **/*.go
---
# Go Coding Guidelines

- Go tests must always use want/got (never expected/actual).
- All checks should use **assert** when the test can continue, and **require** when the test should stop.
- Use **go.uber.org/gomock** for all mocks
