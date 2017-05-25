package core

import (
	"context"
	"math/rand"
	"os"
	"testing"
	"time"

	"golang.org/x/crypto/sha3"

	"github.com/davecgh/go-spew/spew"

	"chain/core/account"
	"chain/core/asset"
	"chain/core/coretest"
	"chain/core/generator"
	"chain/core/pin"
	"chain/core/query"
	"chain/core/txbuilder"
	"chain/core/utxos"
	"chain/crypto/ed25519/chainkd"
	"chain/database/pg/pgtest"
	"chain/encoding/json"
	"chain/errors"
	"chain/protocol/bc"
	"chain/protocol/ivy"
	"chain/protocol/ivy/ivytest"
	"chain/protocol/prottest"
	"chain/protocol/vm"
	"chain/testutil"
)

type key struct {
	pk   []byte
	xpub chainkd.XPub
	path [][]byte
}

func contractArgs(t testing.TB, ctx context.Context, contract *ivy.Contract, clause *ivy.Clause, accounts *account.Manager, assets *asset.Registry) ([]ivy.ContractArg, map[string]interface{}) {
	acc := coretest.CreateAccount(ctx, t, accounts, "", nil)
	var args []ivy.ContractArg
	vals := make(map[string]interface{})
	for _, param := range contract.Params {
		typ := param.InferredType
		if typ == "" {
			typ = param.Type
		}
		switch typ {
		case "PublicKey":
			xpub, pk, path, err := accounts.CreatePubkey(ctx, acc, "")
			if err != nil {
				t.Error(err)
				continue
			}
			args = append(args, ivy.ContractArg{S: (*json.HexBytes)(&pk)})
			vals[param.Name] = &key{pk, xpub, path}
		case "Sha3(PublicKey)":
			xpub, pk, path, err := accounts.CreatePubkey(ctx, acc, "")
			if err != nil {
				t.Error(err)
				continue
			}
			hash := sha3.Sum256(pk)
			hashBytes := hash[:]
			args = append(args, ivy.ContractArg{S: (*json.HexBytes)(&hashBytes)})
			vals[param.Name] = &key{pk, xpub, path}
		case "Program":
			prog, err := accounts.CreateControlProgram(ctx, acc, false, time.Now().Add(time.Minute))
			if err != nil {
				t.Fatal("generating program", err)
			}
			args = append(args, ivy.ContractArg{S: (*json.HexBytes)(&prog)})
			vals[param.Name] = prog
		case "Asset":
			asset := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
			assetBits := asset.Bytes()
			args = append(args, ivy.ContractArg{S: (*json.HexBytes)(&assetBits)})
			vals[param.Name] = asset
		case "Amount":
			amount := rand.Int63()
			args = append(args, ivy.ContractArg{I: &amount})
			vals[param.Name] = amount
		case "Time":
			t := int64(bc.Millis(time.Now().Add(-time.Minute)))
			for _, mt := range clause.MaxTimes {
				if param.Name == mt {
					t = int64(bc.Millis(time.Now().Add(5 * time.Minute)))
					break
				}
			}
			args = append(args, ivy.ContractArg{I: &t})
			vals[param.Name] = t
		case "Sha3(String)":
			bits := make([]byte, 20)
			_, err := rand.Read(bits)
			if err != nil {
				t.Fatal("generating random string")
			}
			hash := sha3.Sum256(bits)
			hashBytes := hash[:]
			args = append(args, ivy.ContractArg{S: (*json.HexBytes)(&hashBytes)})
			vals[param.Name] = bits
		}
	}
	return args, vals
}

func TestContracts(t *testing.T) {
	var (
		_, db     = pgtest.NewDB(t, pgtest.SchemaPath)
		ctx       = context.Background()
		c         = prottest.NewChain(t)
		g         = generator.New(c, nil, db)
		pinStore  = pin.NewStore(db)
		assets    = asset.NewRegistry(db, c, pinStore)
		accounts  = account.NewManager(db, c, pinStore)
		utxoStore = &utxos.Store{DB: db, Chain: c, PinStore: pinStore}
	)
	coretest.CreatePins(ctx, t, pinStore)
	err := pinStore.CreatePin(ctx, utxos.PinName, 0)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	err = pinStore.CreatePin(ctx, utxos.DeletePinName, 0)
	if err != nil {
		testutil.FatalErr(t, err)
	}
	accounts.IndexAccounts(query.NewIndexer(db, c, pinStore))
	go accounts.ProcessBlocks(ctx)
	go utxoStore.ProcessBlocks(ctx)

	asset1 := coretest.CreateAsset(ctx, t, assets, nil, "USD", nil)

	tests := []struct {
		contract string
		clauses  []map[string]string
	}{{
		contract: ivytest.TrivialLock,
		clauses:  nil,
	}, {
		contract: ivytest.LockWithPublicKey,
		clauses:  []map[string]string{{"sig": "publicKey"}},
	}, {
		contract: ivytest.LockWithPKHash,
		clauses:  []map[string]string{{"pubKey": "pubKeyHash", "sig": "pubKeyHash"}},
	}, {
		contract: ivytest.LockWith2of3Keys,
		clauses:  []map[string]string{{"sig1": "pubkey1", "sig2": "pubkey2"}},
	}, {
		contract: ivytest.LockToOutput,
		clauses:  nil,
	}, {
		contract: ivytest.TradeOffer,
		clauses:  []map[string]string{{}, {"sellerSig": "sellerKey"}},
	}, {
		contract: ivytest.EscrowedTransfer,
		clauses:  []map[string]string{{"sig": "agent"}, {"sig": "agent"}},
	}, {
		contract: ivytest.CollateralizedLoan,
		clauses:  nil,
	}, {
		contract: ivytest.RevealPreimage,
		clauses:  []map[string]string{{"string": "hash"}},
	}, {
		contract: ivytest.CallOptionWithSettlement,
		clauses: []map[string]string{
			{"buyerSig": "buyerKey"},
			{},
			{"sellerSig": "sellerKey", "buyerSig": "buyerKey"},
		},
	}}

	for _, test := range tests {
		compiled := compileIvy(compileReq{
			Source: test.contract,
		})
		for i, clause := range compiled.Contracts[0].Clauses {
			args, vals := contractArgs(t, ctx, compiled.Contracts[0], clause, accounts, assets)
			compiled = compileIvy(compileReq{
				Source: test.contract,
				ArgMap: map[string][]ivy.ContractArg{compiled.Contracts[0].Name: args},
			})
			contract := compiled.Contracts[0]
			contractAssetAmount := bc.AssetAmount{AssetId: &asset1, Amount: 1}
			source := txbuilder.Action(assets.NewIssueAction(contractAssetAmount, nil))
			dest := txbuilder.Action(txbuilder.NewControlReceiverAction(
				contractAssetAmount,
				&txbuilder.Receiver{
					ControlProgram: contract.Program,
					ExpiresAt:      time.Now().Add(time.Minute),
				},
				nil,
			))
			tpl, err := txbuilder.Build(ctx, nil, []txbuilder.Action{source, dest}, time.Time{}, time.Now().Add(time.Minute))
			if err != nil {
				t.Log(contract.Name)
				t.Log(clause.Name)
				t.Log(spew.Sdump(compiled))
				t.Error("building locking tpl", err)
				t.Log(errors.Data(err))
				continue
			}
			coretest.SignTxTemplate(t, ctx, tpl, &testutil.TestXPrv)
			err = txbuilder.FinalizeTx(ctx, c, g, tpl.Transaction, tpl.IncludesContract)
			if err != nil {
				t.Log(contract.Name)
				t.Log(clause.Name)
				t.Log(spew.Sdump(compiled))
				t.Error("submitting locking tx", err)
				t.Log(errors.Data(err))
				continue
			}
			b := prottest.MakeBlock(t, c, g.PendingTxs())
			<-pinStore.PinWaiter(utxos.PinName, b.Height)
			<-pinStore.PinWaiter(account.PinName, b.Height)

			source = txbuilder.Action(utxoStore.NewSpendUTXOAction(tpl.Transaction.ResultIds[0], nil))
			actions := []txbuilder.Action{source}
			for _, val := range clause.Values {
				assetAmount := contractAssetAmount
				if val.Name != contract.Value {
					asset := vals[val.Asset].(bc.AssetID)
					assetAmount = bc.AssetAmount{
						AssetId: &asset,
						Amount:  uint64(vals[val.Amount].(int64)),
					}
					actions = append(actions, txbuilder.Action(assets.NewIssueAction(assetAmount, nil)))
				}
				if val.Program != "" {
					actions = append(actions, txbuilder.Action(txbuilder.NewControlReceiverAction(
						assetAmount,
						&txbuilder.Receiver{
							ControlProgram: vals[val.Program].([]byte),
							ExpiresAt:      time.Now().Add(time.Minute),
						},
						nil,
					)))
				} else {
					actions = append(actions, txbuilder.Action(txbuilder.NewRetireAction(assetAmount, nil)))
				}
			}
			tpl, err = txbuilder.Build(ctx, nil, actions, time.Now(), time.Now().Add(time.Minute))
			if err != nil {
				t.Log(contract.Name)
				t.Log(clause.Name)
				t.Log(spew.Sdump(compiled))
				t.Error("building unlocking tpl", err)
				t.Log(errors.Data(err))
				continue
			}
			tpl.IncludesContract = true
			sigInst := &txbuilder.SigningInstruction{}
			for _, arg := range clause.Params {
				switch arg.Type {
				case "Amount":
					amount := rand.Int63()
					sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(vm.Int64Bytes(amount)))
				case "Asset":
					asset := coretest.CreateAsset(ctx, t, assets, nil, "", nil)
					assetBits := asset.Bytes()
					sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(assetBits))
				case "PublicKey":
					valName := test.clauses[i][arg.Name]
					sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(vals[valName].(*key).pk))
				case "Signature":
					valName := test.clauses[i][arg.Name]
					key := vals[valName].(*key)
					var hexPath []json.HexBytes
					for _, v := range key.path {
						hexPath = append(hexPath, v)
					}
					sigInst.WitnessComponents = append(sigInst.WitnessComponents, &txbuilder.RawTxSigWitness{
						Quorum: 1,
						Keys: []txbuilder.KeyID{{
							XPub:           key.xpub,
							DerivationPath: hexPath,
						}},
					})
				case "String":
					valName := test.clauses[i][arg.Name]
					sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(vals[valName].([]byte)))
				}
			}
			if len(contract.Clauses) > 1 {
				sigInst.WitnessComponents = append(sigInst.WitnessComponents, txbuilder.DataWitness(
					vm.Int64Bytes(int64(i)),
				))
			}
			tpl.SigningInstructions[0] = sigInst

			coretest.SignTxTemplate(t, ctx, tpl, &testutil.TestXPrv)
			vm.TraceOut = os.Stdout
			err = txbuilder.FinalizeTx(ctx, c, g, tpl.Transaction, tpl.IncludesContract)
			if err != nil {
				t.Log(contract.Name)
				t.Log(clause.Name)
				t.Log(spew.Sdump(compiled))
				t.Error("submitting unlocking tx", err)
				t.Log(errors.Data(err))
				continue
			}
			b = prottest.MakeBlock(t, c, g.PendingTxs())
			<-pinStore.PinWaiter(utxos.PinName, b.Height)
			<-pinStore.PinWaiter(account.PinName, b.Height)
		}
	}
}
