package asset

import (
	"testing"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/database/pg/pgtest"
	"chain/errors"
)

func TestCreateWallet(t *testing.T) {
	dbtx := pgtest.TxWithSQL(t, `
		INSERT INTO projects (id, name) VALUES ('app-id-0', 'app-0');
	`)
	defer dbtx.Rollback()
	ctx := pg.NewContext(context.Background(), dbtx)

	wallet, err := CreateWallet(ctx, "app-id-0", &CreateWalletRequest{Label: "foo", GenerateKey: true})
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}
	if wallet.ID == "" {
		t.Errorf("got empty wallet id")
	}
	var valid bool
	const checkQ = `
		SELECT SUBSTR(generated_keys[1], 1, 4)='xprv' FROM manager_nodes LIMIT 1
	`
	err = dbtx.QueryRow(checkQ).Scan(&valid)
	if err != nil {
		t.Fatal(err)
	}
	if !valid {
		t.Errorf("private key not stored")
	}
}

func TestNewKey(t *testing.T) {
	pub, priv, err := newKey()
	if err != nil {
		t.Log(errors.Stack(err))
		t.Fatal(err)
	}

	validPub, err := priv.Neuter()
	if err != nil {
		t.Fatal(err)
	}

	if validPub.String() != pub.String() {
		t.Fatal("incorrect private/public key pair")
	}
}
