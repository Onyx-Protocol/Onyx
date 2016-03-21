package voting

import (
	"fmt"

	"golang.org/x/net/context"

	"chain/database/pg"
	"chain/errors"
	"chain/fedchain/bc"
)

// RightWithUTXO encapsulates a claim to a right token and the current UTXO
// of the right token.
//
// UTXO      — The current utxo for this voting right. Any transactions
//             involving this voting right token need to consume this utxo.
// Outpoint  - The outpoint recording the account's claim to the voting right
//             token. If the Outpoint equals the UTXO, then this voting right
//             claim is the current holder. Otherwise, this claim doesn't
//             currently hold the voting right but may recall the claim by
//             spending the UTXO and invoking the recall clause in the
//             sigscript.
// AssetID   - The asset ID of the voting right token.
// AccountID - The account id that has a claim to the voting right token. This
//             may be nil if it's an account on another node.
//
type RightWithUTXO struct {
	UTXO      bc.Outpoint
	Outpoint  bc.Outpoint
	AssetID   bc.AssetID
	AccountID *string
	rightScriptData
}

func insertVotingRight(ctx context.Context, assetID bc.AssetID, outpoint bc.Outpoint, data rightScriptData) error {
	db, ctx, err := pg.Begin(ctx)
	if err != nil {
		return errors.Wrap(err, "opening database tx")
	}
	defer db.Rollback(ctx)

	const q = `
		INSERT INTO voting_right_txs
			(asset_id, account_id, tx_hash, index, holder, deadline, delegatable, ownership_chain)
			VALUES($1, (SELECT account_id FROM addresses WHERE pk_script=$4), $2, $3, $4, $5, $6, $7)
	`
	_, err = pg.Exec(ctx, q, assetID, outpoint.Hash, outpoint.Index, data.HolderScript,
		data.Deadline, data.Delegatable, data.OwnershipChain[:])
	if err != nil {
		return errors.Wrap(err, "inserting into voting_right_txs")
	}
	return errors.Wrap(db.Commit(ctx), "commiting database tx")
}

type votingRightsQuery struct {
	outpoint *bc.Outpoint
}

func (q votingRightsQuery) Where() (string, []interface{}) {
	var (
		whereClause string
		values      []interface{}
		param       int = 1
	)

	// TODO(jackson): Add additional query parameters.
	if q.outpoint != nil {
		whereClause = fmt.Sprintf("%s AND vr.tx_hash = $%d AND vr.index = $%d\n", whereClause, param, param+1)
		values = append(values, q.outpoint.Hash, q.outpoint.Index)
		param += 2
	}
	return whereClause, values
}

// FindRightForOutpoint returns the voting right with the provided tx outpoint.
func FindRightForOutpoint(ctx context.Context, out bc.Outpoint) (*RightWithUTXO, error) {
	rights, err := findVotingRights(ctx, votingRightsQuery{outpoint: &out})
	if err != nil {
		return nil, err
	}
	if len(rights) != 1 {
		return nil, fmt.Errorf("expected 1 right, found %d", len(rights))
	}
	return rights[0], nil
}

func findVotingRights(ctx context.Context, q votingRightsQuery) ([]*RightWithUTXO, error) {
	var (
		results []*RightWithUTXO
	)

	const sqlQ = `
		SELECT
			u.tx_hash AS utxo_hash,
			u.index   AS utxo_index,
			vr.tx_hash,
			vr.index,
			vr.asset_id,
			vr.holder,
			vr.deadline,
			vr.delegatable,
			vr.ownership_chain
		FROM voting_right_txs vr
		INNER JOIN utxos u ON vr.asset_id = u.asset_id
		WHERE
			u.asset_id = vr.asset_id AND
			NOT EXISTS (SELECT 1 FROM pool_inputs pi WHERE pi.tx_hash = u.tx_hash AND pi.index = u.index)
	`
	sqlSuffix, values := q.Where()

	rows, err := pg.Query(ctx, sqlQ+sqlSuffix, values...)
	if err != nil {
		return nil, errors.Wrap(err, "query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			right          RightWithUTXO
			ownershipChain []byte
		)

		err = rows.Scan(
			&right.UTXO.Hash, &right.UTXO.Index,
			&right.Outpoint.Hash, &right.Outpoint.Index, &right.AssetID,
			&right.HolderScript, &right.Deadline, &right.Delegatable, &ownershipChain)
		if err != nil {
			return nil, errors.Wrap(err, "scanning RightWithUTXO")
		}
		copy(right.OwnershipChain[:], ownershipChain)
		results = append(results, &right)
	}

	if err = rows.Err(); err != nil {
		return nil, errors.Wrap(err, "end scan")
	}
	return results, nil
}
