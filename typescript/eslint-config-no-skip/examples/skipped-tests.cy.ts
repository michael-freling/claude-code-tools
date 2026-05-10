// Example Cypress file to demonstrate ESLint catching skipped tests
// Run: pnpm lint:examples

describe('Cypress example test suite', () => {
  it('normal test should pass lint', () => {
    cy.visit('/');
  });

  // These should be caught by eslint-plugin-mocha's no-skipped-tests rule:
  it.skip('skipped test with it.skip', () => {
    cy.visit('/');
  });

  context.skip('skipped context block', () => {
    it('test inside skipped context', () => {
      cy.visit('/');
    });
  });
});

describe.skip('skipped describe block', () => {
  it('test inside skipped describe', () => {
    cy.visit('/');
  });
});

// These should be caught by eslint-plugin-mocha's no-exclusive-tests rule:
describe.only('exclusive describe block', () => {
  it('test inside only describe', () => {
    cy.visit('/');
  });
});

it.only('exclusive test with it.only', () => {
  cy.visit('/');
});
