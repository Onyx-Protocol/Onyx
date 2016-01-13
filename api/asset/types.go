package asset

import (
	"time"

	"golang.org/x/net/context"

	"chain/api/txdb"
	"chain/fedchain/bc"
)

type (
	ReserveResultItem struct {
		TxInput       *bc.TxInput
		TemplateInput *Input
	}

	ReserveResult struct {
		Items  []*ReserveResultItem
		Change *Destination
	}

	Reserver interface {
		Reserve(context.Context, *bc.AssetAmount, time.Duration) (*ReserveResult, error)
	}

	// A Source is a source of funds for a transaction.
	Source struct {
		bc.AssetAmount
		Reserver Reserver
	}

	Receiver interface {
		IsChange() bool
		PKScript() []byte
		// Make sure the UTXOInserter list contains the right kind of
		// UTXOInserter (adding one if necessary), and add data about the
		// txoutput to it.
		AccumulateUTXO(context.Context, *bc.Outpoint, *bc.TxOutput, []UTXOInserter) ([]UTXOInserter, error)
		MarshalJSON() ([]byte, error)
	}

	// A Destination is a payment destination for a transaction.
	Destination struct {
		bc.AssetAmount
		IsChange bool
		Metadata []byte
		Receiver Receiver
	}

	UTXOInserter interface {
		// This function performs UTXO insertion into the db.  It's called
		// as one of the final steps in FinalizeTx().  There may be many
		// UTXOInserters, each inserting utxos of a different type.
		InsertUTXOs(context.Context) ([]*txdb.Output, error)
	}
)

func (source *Source) Reserve(ctx context.Context, ttl time.Duration) (*ReserveResult, error) {
	return source.Reserver.Reserve(ctx, &source.AssetAmount, ttl)
}

func (dest *Destination) PKScript() []byte { return dest.Receiver.PKScript() }
