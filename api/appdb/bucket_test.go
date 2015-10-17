package appdb

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
)

func TestCreateBucket(t *testing.T) {
	withContext(t, sampleProjectFixture, func(t *testing.T, ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, "proj-id-0", "foo")
		bucket, err := CreateBucket(ctx, managerNode.ID, "foo")
		if err != nil {
			t.Error("unexpected error", err)
		}
		if bucket == nil || bucket.ID == "" {
			t.Error("got nil bucket or empty id")
		}
		if bucket.Label != "foo" {
			t.Errorf("label = %q want foo", bucket.Label)
		}
	})
}

func TestCreateBucketBadLabel(t *testing.T) {
	withContext(t, sampleProjectFixture, func(t *testing.T, ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, "proj-id-0", "foo")
		_, err := CreateBucket(ctx, managerNode.ID, "")
		if err == nil {
			t.Error("err = nil, want error")
		}
	})
}

func TestBucketBalance(t *testing.T) {
	const sql = `
		INSERT INTO utxos (txid, index, asset_id, amount, addr_index, account_id, manager_node_id)
		VALUES ('t0', 0, 'a1', 10, 0, 'b0', 'mn1'),
		       ('t1', 1, 'a1', 5, 0, 'b0', 'mn1'),
		       ('t2', 2, 'a2', 20, 0, 'b0', 'mn1');
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		cases := []struct {
			bID      string
			prev     string
			limit    int
			want     []*Balance
			wantLast string
		}{{
			bID:      "b0",
			limit:    5,
			want:     []*Balance{{"a1", 15, 15}, {"a2", 20, 20}},
			wantLast: "a2",
		}, {
			bID:      "b0",
			prev:     "a1",
			limit:    5,
			want:     []*Balance{{"a2", 20, 20}},
			wantLast: "a2",
		}, {
			bID:      "b0",
			prev:     "a2",
			limit:    5,
			want:     nil,
			wantLast: "",
		}, {
			bID:      "b0",
			limit:    1,
			want:     []*Balance{{"a1", 15, 15}},
			wantLast: "a1",
		}, {
			bID:      "nonexistent",
			limit:    5,
			want:     nil,
			wantLast: "",
		}}

		for _, c := range cases {
			got, gotLast, err := BucketBalance(ctx, c.bID, c.prev, c.limit)
			if err != nil {
				t.Errorf("BucketBalance(%s, %s, %d): unexpected error %v", c.bID, c.prev, c.limit, err)
				continue
			}

			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("BucketBalance(%s, %s, %d) = %v want %v", c.bID, c.prev, c.limit, got, c.want)
			}

			if gotLast != c.wantLast {
				t.Errorf("BucketBalance(%s, %s, %d) last = %v want %v", c.bID, c.prev, c.limit, gotLast, c.wantLast)
			}
		}
	})
}

func TestListBuckets(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0'),
			('manager-node-id-1', 'proj-id-0', 1, 'manager-node-1');

		INSERT INTO accounts (id, manager_node_id, key_index, label) VALUES
			('bucket-id-0', 'manager-node-id-0', 0, 'bucket-0'),
			('bucket-id-1', 'manager-node-id-0', 1, 'bucket-1'),
			('bucket-id-2', 'manager-node-id-1', 2, 'bucket-2'),
			('bucket-id-3', 'manager-node-id-0', 3, 'bucket-3');
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			managerNodeID string
			prev          string
			limit         int
			want          []*Bucket
			wantLast      string
		}{
			{
				managerNodeID: "manager-node-id-0",
				limit:         5,
				want: []*Bucket{
					{ID: "bucket-id-3", Label: "bucket-3", Index: []uint32{0, 3}},
					{ID: "bucket-id-1", Label: "bucket-1", Index: []uint32{0, 1}},
					{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
				},
				wantLast: "bucket-id-0",
			},
			{
				managerNodeID: "manager-node-id-1",
				limit:         5,
				want: []*Bucket{
					{ID: "bucket-id-2", Label: "bucket-2", Index: []uint32{0, 2}},
				},
				wantLast: "bucket-id-2",
			},
			{
				managerNodeID: "nonexistent",
				want:          nil,
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				want: []*Bucket{
					{ID: "bucket-id-3", Label: "bucket-3", Index: []uint32{0, 3}},
					{ID: "bucket-id-1", Label: "bucket-1", Index: []uint32{0, 1}},
				},
				wantLast: "bucket-id-1",
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				prev:          "bucket-id-1",
				want: []*Bucket{
					{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
				},
				wantLast: "bucket-id-0",
			},
			{
				managerNodeID: "manager-node-id-0",
				limit:         2,
				prev:          "bucket-id-0",
				want:          nil,
				wantLast:      "",
			},
		}

		for _, ex := range examples {
			got, gotLast, err := ListBuckets(ctx, ex.managerNodeID, ex.prev, ex.limit)
			if err != nil {
				t.Fatal(err)
			}

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("ListBuckets(%v, %v, %d):\ngot:  %v\nwant: %v", ex.managerNodeID, ex.prev, ex.limit, got, ex.want)
			}

			if gotLast != ex.wantLast {
				t.Errorf("ListBuckets(%v, %v, %d):\ngot last:  %v\nwant last: %v",
					ex.managerNodeID, ex.prev, ex.limit, gotLast, ex.wantLast)
			}
		}
	})
}

func TestGetBucket(t *testing.T) {
	const sql = `
		INSERT INTO projects (id, name) VALUES
			('proj-id-0', 'proj-0');

		INSERT INTO manager_nodes (id, project_id, key_index, label) VALUES
			('manager-node-id-0', 'proj-id-0', 0, 'manager-node-0');

		INSERT INTO accounts (id, manager_node_id, key_index, label) VALUES
			('bucket-id-0', 'manager-node-id-0', 0, 'bucket-0')
	`
	withContext(t, sql, func(t *testing.T, ctx context.Context) {
		examples := []struct {
			id      string
			want    *Bucket
			wantErr error
		}{
			{
				"bucket-id-0",
				&Bucket{ID: "bucket-id-0", Label: "bucket-0", Index: []uint32{0, 0}},
				nil,
			},
			{
				"nonexistent",
				nil,
				pg.ErrUserInputNotFound,
			},
		}

		for _, ex := range examples {
			t.Log("Example:", ex.id)

			got, gotErr := GetBucket(ctx, ex.id)

			if !reflect.DeepEqual(got, ex.want) {
				t.Errorf("bucket:\ngot:  %v\nwant: %v", got, ex.want)
			}

			if errors.Root(gotErr) != ex.wantErr {
				t.Errorf("get bucket error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
			}
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	withContext(t, sampleProjectFixture, func(t *testing.T, ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, "proj-id-0", "foo")
		account, err := CreateBucket(ctx, managerNode.ID, "foo")
		if err != nil {
			t.Error("unexpected error", err)
		}
		if account == nil || account.ID == "" {
			t.Error("got nil account or empty id")
		}
		if account.Label != "foo" {
			t.Errorf("label = %q want foo", account.Label)
		}

		newLabel := "bar"
		err = UpdateAccount(ctx, account.ID, &newLabel)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		account, err = GetBucket(ctx, account.ID)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}
		if account.Label != newLabel {
			t.Errorf("expected %s, got %s", newLabel, account.Label)
		}
	})
}

// Test that calling UpdateManagerNode with no new label is a no-op.
func TestUpdateAccountNoUpdate(t *testing.T) {
	withContext(t, sampleProjectFixture, func(t *testing.T, ctx context.Context) {
		managerNode := newTestManagerNode(t, ctx, "proj-id-0", "foo")
		account, err := CreateBucket(ctx, managerNode.ID, "foo")
		if err != nil {
			t.Fatalf("could not create account: %v", err)
		}
		if account == nil {
			t.Fatal("could not create account (got nil)")
		}
		if account.ID == "" {
			t.Fatal("got empty id when creating account")
		}
		if account.Label != "foo" {
			t.Fatalf("wrong label when creating account, expected foo, got %q", account.Label)
		}

		err = UpdateAccount(ctx, account.ID, nil)
		if err != nil {
			t.Errorf("unexpected error %v", err)
		}

		account, err = GetBucket(ctx, account.ID)
		if err != nil {
			t.Fatalf("could not get account with id %s", account.ID)
		}
		if account.Label != "foo" {
			t.Errorf("expected foo, got %s", account.Label)
		}
	})
}
