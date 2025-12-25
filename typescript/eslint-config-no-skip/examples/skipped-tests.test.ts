// Example file to demonstrate ESLint catching skipped tests
// Run: pnpm lint:examples

describe('example test suite', () => {
  it('normal test should pass lint', () => {
    expect(true).toBe(true);
  });

  // These should be caught by eslint-plugin-jest's no-disabled-tests rule:
  it.skip('skipped test with it.skip', () => {
    expect(true).toBe(true);
  });

  test.skip('skipped test with test.skip', () => {
    expect(true).toBe(true);
  });

  xit('skipped test with xit', () => {
    expect(true).toBe(true);
  });

  xtest('skipped test with xtest', () => {
    expect(true).toBe(true);
  });
});

describe.skip('skipped describe block', () => {
  it('test inside skipped describe', () => {
    expect(true).toBe(true);
  });
});

xdescribe('xdescribe block', () => {
  it('test inside xdescribe', () => {
    expect(true).toBe(true);
  });
});
