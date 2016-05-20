package appdb_test

import (
	"reflect"
	"testing"

	"golang.org/x/net/context"

	. "chain/core/appdb"
	"chain/core/asset/assettest"
	"chain/cos/hdkey"
	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestInsertIssuerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	newTestIssuerNode(t, ctx, nil, "foo")
}

func TestInsertIssuerNodeIdempotence(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	project1 := newTestProject(t, ctx, "project-1", nil)
	project2 := newTestProject(t, ctx, "project-2", newTestUser(t, ctx, "two@user.com", "password"))

	idempotencyKey := "my-issuer-node-client-token"
	in1, err := InsertIssuerNode(ctx, project1.ID, "issuer-node", []*hdkey.XKey{dummyXPub}, nil, 1, &idempotencyKey)
	if err != nil {
		t.Fatalf("could not create issuer node: %v", err)
	}
	if in1.ID == "" {
		t.Fatal("got empty issuer node id")
	}

	in2, err := InsertIssuerNode(ctx, project1.ID, "issuer-node", []*hdkey.XKey{dummyXPub}, nil, 1, &idempotencyKey)
	if err != nil {
		t.Fatalf("failed on 2nd call to insert issuer node: %s", err)
	}
	if !reflect.DeepEqual(in1, in2) {
		t.Errorf("got=%#v\nwant=%#v", in2, in1)
	}

	in3, err := InsertManagerNode(ctx, project2.ID, "issuer-node", []*hdkey.XKey{dummyXPub}, nil, 0, 1, &idempotencyKey)
	if err != nil {
		t.Fatalf("failed on 3rd call to insert issuer node: %s", err)
	}
	if in3.ID == in1.ID {
		t.Error("client_token should be project-scoped")
	}

	newIdempotencyKey := "my-new-issuer-node"
	in4, err := InsertManagerNode(ctx, project1.ID, "issuer-node", []*hdkey.XKey{dummyXPub}, nil, 0, 1, &newIdempotencyKey)
	if err != nil {
		t.Fatalf("failed on 4th call to insert issuer node: %s", err)
	}
	if in4.ID == in1.ID {
		t.Errorf("got=%#v want new issuer node, not %#v", in4, in1)
	}
}

func TestListIssuerNodes(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	proj0ID := assettest.CreateProjectFixture(ctx, t, "", "proj-0")
	proj1ID := assettest.CreateProjectFixture(ctx, t, "", "proj-1")

	in0ID := assettest.CreateIssuerNodeFixture(ctx, t, proj0ID, "in-0", nil, nil)
	in1ID := assettest.CreateIssuerNodeFixture(ctx, t, proj0ID, "in-1", nil, nil)
	in2ID := assettest.CreateIssuerNodeFixture(ctx, t, proj1ID, "in-2", nil, nil)
	assettest.CreateArchivedIssuerNodeFixture(ctx, t, proj0ID, "in-3", nil, nil)

	examples := []struct {
		projID string
		want   []*IssuerNode
	}{
		{
			proj0ID,
			[]*IssuerNode{
				{ID: in0ID, Label: "in-0"},
				{ID: in1ID, Label: "in-1"},
			},
		},
		{
			proj1ID,
			[]*IssuerNode{
				{ID: in2ID, Label: "in-2"},
			},
		},
	}

	for _, ex := range examples {
		t.Log("Example:", ex.projID)

		got, err := ListIssuerNodes(ctx, ex.projID)
		if err != nil {
			t.Fatal(err)
		}

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("issuer nodes:\ngot:  %+v\nwant: %+v", got[0], ex.want[0])
		}
	}
}

func TestGetIssuerNodes(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	proj := newTestProject(t, ctx, "foo", nil)
	in, err := InsertIssuerNode(ctx, proj.ID, "in-0", []*hdkey.XKey{dummyXPub}, []*hdkey.XKey{dummyXPrv}, 1, nil)
	if err != nil {
		t.Fatalf("unexpected error on InsertIssuerNode: %v", err)
	}
	examples := []struct {
		id      string
		want    *IssuerNode
		wantErr error
	}{
		{
			in.ID,
			&IssuerNode{
				ID:    in.ID,
				Label: "in-0",
				Keys: []*NodeKey{
					{
						Type: "node",
						XPub: dummyXPub,
						XPrv: dummyXPrv,
					},
				},
				SigsReqd: 1,
			},
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

		got, gotErr := GetIssuerNode(ctx, ex.id)

		if !reflect.DeepEqual(got, ex.want) {
			t.Errorf("issuer node:\ngot:  %v\nwant: %v", got, ex.want)
		}

		if errors.Root(gotErr) != ex.wantErr {
			t.Errorf("get issuer node error:\ngot:  %v\nwant: %v", errors.Root(gotErr), ex.wantErr)
		}
	}
}

func TestUpdateIssuerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))
	issuerNode := newTestIssuerNode(t, ctx, nil, "foo")

	newLabel := "bar"

	err := UpdateIssuerNode(ctx, issuerNode.ID, &newLabel)
	if err != nil {
		t.Errorf("update issuer node error %v", err)
	}

	issuerNode, err = GetIssuerNode(ctx, issuerNode.ID)
	if err != nil {
		t.Fatalf("could not get issuer node with id %s: %v", issuerNode.ID, err)
	}
	if issuerNode.Label != newLabel {
		t.Errorf("expected %s, got %s", newLabel, issuerNode.Label)
	}
}

// Test that calling UpdateIssuerNode with no new label is a no-op.
func TestUpdateIssuerNodeNoUpdate(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	issuerNode := newTestIssuerNode(t, ctx, nil, "foo")
	err := UpdateIssuerNode(ctx, issuerNode.ID, nil)
	if err != nil {
		t.Errorf("update issuer node error %v", err)
	}

	issuerNode, err = GetIssuerNode(ctx, issuerNode.ID)
	if err != nil {
		t.Fatalf("could not get issuer node with id %s: %v:", issuerNode.ID, err)
	}
	if issuerNode.Label != "foo" {
		t.Errorf("expected foo, got %s", issuerNode.Label)
	}
}

func TestArchiveIssuerNode(t *testing.T) {
	ctx := pg.NewContext(context.Background(), pgtest.NewTx(t))

	issuerNode := newTestIssuerNode(t, ctx, nil, "foo")
	asset := newTestAsset(t, ctx, issuerNode)
	err := ArchiveIssuerNode(ctx, issuerNode.ID)
	if err != nil {
		t.Errorf("could not archive issuer node with id %s: %v", issuerNode.ID, err)
	}

	var archived bool
	checkQ := `SELECT archived FROM issuer_nodes WHERE id = $1`
	err = pg.QueryRow(ctx, checkQ, issuerNode.ID).Scan(&archived)

	if !archived {
		t.Errorf("expected issuer node %s to be archived", issuerNode.ID)
	}

	checkAssetQ := `SELECT archived FROM assets WHERE id = $1`
	err = pg.QueryRow(ctx, checkAssetQ, asset.Hash.String()).Scan(&archived)
	if !archived {
		t.Errorf("expected child asset %s to be archived", asset.Hash.String())
	}
}
