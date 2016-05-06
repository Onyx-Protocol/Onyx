/*

Package pgtest provides support functions for tests that need to
use Postgres. Most clients will just call NewTx or NewContext;
those that need more control can start a DB directly.

    func TestSimple(t *testing.T) {
        dbtx := pgtest.NewTx(t)
        ...
    }

    func TestComplex(t *testing.T) {
        ctx := pgtest.NewContext(t)
        ...
        dbtx, ctx, err := pg.Begin(ctx)
        ...
    }

Prefer NewTx when the caller (usually a test function)
can run in exactly one transaction.
It's significantly faster than NewContext.

*/
package pgtest
