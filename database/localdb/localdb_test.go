package localdb

import (
	"testing"
)

func TestRestartDB(t *testing.T) {
	t.Log("hi")
	// rocksDir, err := ioutil.TempDir("", "rocks_testdb")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer os.RemoveAll(rocksDir)

	// // Create a new fresh db and write... something.
	// ldb1, err := Open(rocksDir)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer ldb1.Close()

	// testItem := &localdbtest.TestItem{Value: "bar"}
	// err = ldb1.Put("foo", testItem)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// ldb1.Close()

	// // Re-open the database and verify that the write is still there.
	// ldb2, err := Open(rocksDir)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// defer ldb2.Close()

	// resultItem := new(localdbtest.TestItem)
	// err = ldb2.Get("foo", resultItem)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// if resultItem.Value != "bar" {
	// 	t.Fatalf("expected value read to be 'bar', got %s", resultItem.Value)
	// }
}
