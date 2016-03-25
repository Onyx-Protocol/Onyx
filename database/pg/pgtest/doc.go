/*

Package pgtest provides support functions for tests that need to
use Postgres. Most clients will just call NewContext; those
that need more control can start a DB directly.

    func TestFoo(t *testing.T) {
        ctx := pgtest.NewContext(t)
        ...
    }

*/
package pgtest
