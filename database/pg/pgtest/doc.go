/*

Package pgtest provides support functions for tests that need to
use Postgres. Most clients will just call NewTx or NewDB;
those that need more control can start a DB directly.

    func TestSimple(t *testing.T) {
        dbtx := pgtest.NewTx(t)
        ...
    }

    func TestComplex(t *testing.T) {
        _, db := pgtest.NewDB(t, pgtest.SchemaPath)
        ...
        dbtx, err := db.Begin(ctx)
        ...
    }

Prefer NewTx when the caller (usually a test function)
can run in exactly one transaction.
It's significantly faster than NewDB.

*/
package pgtest
