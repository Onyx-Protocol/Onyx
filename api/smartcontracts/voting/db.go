package voting

import (
	"database/sql"
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"chain/cos/bc"
	"chain/database/pg"
	"chain/errors"
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
	UTXO         bc.Outpoint
	Outpoint     bc.Outpoint
	BlockHeight  uint64
	BlockTxIndex int
	AssetID      bc.AssetID
	AccountID    *string
	rightScriptData
}

// Token describes the state of a voting token. It's scoped to a particular
// voting right and agenda item.
type Token struct {
	AssetID   bc.AssetID
	Outpoint  bc.Outpoint
	Amount    int64
	AccountID string
	tokenScriptData
}

type cursor struct {
	prevBlockHeight uint64
	prevBlockPos    int
}

func (c cursor) String() string {
	if c.prevBlockHeight == 0 && c.prevBlockPos == 0 {
		return ""
	}
	return fmt.Sprintf("%d-%d", c.prevBlockHeight, c.prevBlockPos)
}

func insertVotingRight(ctx context.Context, assetID bc.AssetID, blockHeight uint64, blockTxIndex int, outpoint bc.Outpoint, data rightScriptData) error {
	const q = `
		INSERT INTO voting_right_txs
			(asset_id, account_id, tx_hash, index, block_height, block_tx_index, holder, deadline, delegatable, ownership_chain, admin_script)
			VALUES($1, (SELECT account_id FROM addresses WHERE pk_script=$6), $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (tx_hash, index) DO NOTHING
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, assetID, outpoint.Hash, outpoint.Index, blockHeight, blockTxIndex,
		data.HolderScript, data.Deadline, data.Delegatable, data.OwnershipChain[:], data.AdminScript)
	return errors.Wrap(err, "inserting into voting_right_txs")
}

func upsertVotingToken(ctx context.Context, assetID bc.AssetID, outpoint bc.Outpoint, amount uint64, data tokenScriptData) error {
	const q = `
		INSERT INTO voting_tokens
			(asset_id, right_asset_id, tx_hash, index, state, closed, vote, option_count, secret_hash, admin_script, amount)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (asset_id, right_asset_id) DO UPDATE
		  SET tx_hash = $3, index = $4, state = $5, closed = $6, vote = $7, secret_hash = $9
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, assetID, data.Right,
		outpoint.Hash, outpoint.Index, data.State.Base(), data.State.Finished(),
		data.Vote, data.OptionCount, data.SecretHash, data.AdminScript, amount)
	return errors.Wrap(err, "upserting into voting_tokens")
}

// voidRecalledVotingRights takes the outpoint of the contract being executed
// and an ownership hash in the active chain of ownership. It then voids
// all voting right claims back to—and including—the voting right with
// the provided ownership hash.
func voidRecalledVotingRights(ctx context.Context, out bc.Outpoint, ownershipHash bc.Hash) error {
	const q = `
		WITH right_token AS (
			SELECT asset_id
			FROM voting_right_txs
			WHERE tx_hash = $1 AND index = $2
			LIMIT 1
		),
		recall_point AS (
			SELECT block_height, block_tx_index, asset_id
			FROM voting_right_txs
			WHERE asset_id = (SELECT asset_id FROM right_token) AND ownership_chain = $3 AND NOT void
			LIMIT 1
		)
		UPDATE voting_right_txs SET void = 't'
		FROM recall_point rp
		WHERE voting_right_txs.asset_id = rp.asset_id
		AND (voting_right_txs.block_height, voting_right_txs.block_tx_index) >= (rp.block_height, rp.block_tx_index)
	`
	res, err := pg.FromContext(ctx).Exec(ctx, q, out.Hash, out.Index, ownershipHash[:])
	if err != nil {
		return errors.Wrap(err, "voiding voting_right_txs")
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "rows affected while voiding")
	}
	if affected < 1 {
		return fmt.Errorf("at least one voting right should be voided: %s, %d : %x", out.Hash, out.Index, ownershipHash[:])
	}
	return nil
}

// voidVotingRight takes the outpoint of a voting right claim
// and marks it as void.
func voidVotingRight(ctx context.Context, prev bc.Outpoint) error {
	const q = `
		UPDATE voting_right_txs SET void = 't'
		WHERE tx_hash = $1 AND index = $2
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, prev.Hash, prev.Index)
	return errors.Wrap(err, "voiding voting_right_txs")
}

type votingRightsQuery struct {
	accountID string
	outpoint  *bc.Outpoint
	assetID   *bc.AssetID
	utxoOnly  bool

	cursor *cursor
	limit  int
}

func (q votingRightsQuery) Limit() string {
	if q.limit == 0 {
		return ""
	}
	return fmt.Sprintf(" LIMIT %d", q.limit)
}

func (q votingRightsQuery) Where() (string, []interface{}) {
	var (
		whereClause string
		values      []interface{}
		param       int = 1
	)

	if q.accountID != "" {
		whereClause = fmt.Sprintf("%s AND vr.account_id = $%d\n", whereClause, param)
		values = append(values, q.accountID)
		param++
	}
	if q.outpoint != nil {
		whereClause = fmt.Sprintf("%s AND vr.tx_hash = $%d AND vr.index = $%d\n", whereClause, param, param+1)
		values = append(values, q.outpoint.Hash, q.outpoint.Index)
		param += 2
	}
	if q.assetID != nil {
		whereClause = fmt.Sprintf("%s AND vr.asset_id = $%d\n", whereClause, param)
		values = append(values, *q.assetID)
		param++
	}
	if q.utxoOnly {
		whereClause = whereClause + " AND (vr.tx_hash, vr.index) = (u.tx_hash, u.index)"
	}
	if q.cursor != nil {
		whereClause = fmt.Sprintf("%s AND (vr.block_height, vr.block_tx_index) > ($%d, $%d)\n", whereClause, param, param+1)
		values = append(values, q.cursor.prevBlockHeight, q.cursor.prevBlockPos)
		param += 2
	}
	whereClause = fmt.Sprintf("%s AND vr.void = 'f'\n", whereClause)
	return whereClause, values
}

// FindRightsForAccount returns all voting rights belonging to the provided account.
func FindRightsForAccount(ctx context.Context, accountID string, prev string, limit int) ([]*RightWithUTXO, map[bc.AssetID]string, string, error) {
	// Since the sort criteria is composite, the cursor is composite.
	var (
		prevBlockHeight uint64
		prevBlockPos    int
		cur             *cursor
	)
	_, err := fmt.Sscanf(prev, "%d-%d", &prevBlockHeight, &prevBlockPos)

	// ignore malformed cursors
	if err == nil {
		cur = &cursor{
			prevBlockHeight: prevBlockHeight,
			prevBlockPos:    prevBlockPos,
		}
	}

	rights, next, err := findVotingRights(ctx, votingRightsQuery{
		accountID: accountID,
		cursor:    cur,
		limit:     limit,
	})
	if err != nil {
		return nil, nil, "", err
	}

	var assets []string
	for _, right := range rights {
		assets = append(assets, right.AssetID.String())
	}
	const holderQ = `
		SELECT vr.asset_id, vr.account_id
		FROM voting_right_txs vr
		JOIN utxos u ON (vr.tx_hash, vr.index) = (u.tx_hash, u.index)
		WHERE (vr.tx_hash, vr.index) NOT IN (TABLE pool_inputs)
		AND vr.asset_id=ANY($1::text[])
	`
	holderMap := make(map[bc.AssetID]string)
	err = pg.ForQueryRows(ctx, holderQ, pg.Strings(assets), func(asset bc.AssetID, account string) {
		holderMap[asset] = account
	})
	if err != nil {
		return nil, nil, "", err
	}

	return rights, holderMap, next, nil
}

// FindRightForOutpoint returns the voting right with the provided tx outpoint.
func FindRightForOutpoint(ctx context.Context, out bc.Outpoint) (*RightWithUTXO, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{outpoint: &out})
	if err != nil {
		return nil, err
	}
	if len(rights) != 1 {
		return nil, fmt.Errorf("expected 1 right, found %d", len(rights))
	}
	return rights[0], nil
}

// FindRightsForAsset return all non-void claims to the voting right
// token with the provided asset ID. The resulting voting rights will
// be sorted chronologically (by block_height, block_tx_index). Effectively,
// this function returns the entire active chain of ownership for the
// voting right token.
func FindRightsForAsset(ctx context.Context, assetID bc.AssetID) ([]*RightWithUTXO, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{assetID: &assetID})
	if err != nil {
		return nil, err
	}
	return rights, nil
}

// FindRightUTXO looks up the current utxo for the voting right with the
// provided assetID.
func FindRightUTXO(ctx context.Context, assetID bc.AssetID) (*RightWithUTXO, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{
		assetID:  &assetID,
		utxoOnly: true,
	})
	if err != nil {
		return nil, err
	}
	if len(rights) == 0 {
		return nil, pg.ErrUserInputNotFound
	} else if len(rights) != 1 {
		return nil, fmt.Errorf("expected 1 right, found %d", len(rights))
	}
	return rights[0], nil
}

func findVotingRights(ctx context.Context, q votingRightsQuery) ([]*RightWithUTXO, string, error) {
	var (
		cur     cursor
		results []*RightWithUTXO
	)

	const sqlQ = `
		SELECT
			u.tx_hash AS utxo_hash,
			u.index   AS utxo_index,
			vr.tx_hash,
			vr.index,
			vr.block_height,
			vr.block_tx_index,
			vr.asset_id,
			vr.account_id,
			vr.holder,
			vr.deadline,
			vr.delegatable,
			vr.ownership_chain,
			vr.admin_script
		FROM voting_right_txs vr
		INNER JOIN utxos u ON vr.asset_id = u.asset_id
		WHERE
			u.asset_id = vr.asset_id AND
			NOT EXISTS (SELECT 1 FROM pool_inputs pi WHERE pi.tx_hash = u.tx_hash AND pi.index = u.index)
	`
	whereSQL, values := q.Where()
	queryStr := fmt.Sprintf("%s%s ORDER BY vr.block_height ASC, vr.block_tx_index ASC%s", sqlQ, whereSQL, q.Limit())
	rows, err := pg.Query(ctx, queryStr, values...)
	if err != nil {
		return nil, "", errors.Wrap(err, "query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			right          RightWithUTXO
			ownershipChain []byte
		)

		err = rows.Scan(
			&right.UTXO.Hash, &right.UTXO.Index,
			&right.Outpoint.Hash, &right.Outpoint.Index,
			&right.BlockHeight, &right.BlockTxIndex,
			&right.AssetID, &right.AccountID,
			&right.HolderScript, &right.Deadline, &right.Delegatable, &ownershipChain,
			&right.AdminScript)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning RightWithUTXO")
		}
		copy(right.OwnershipChain[:], ownershipChain)
		results = append(results, &right)
		cur = cursor{
			prevBlockHeight: right.BlockHeight,
			prevBlockPos:    right.BlockTxIndex,
		}
	}

	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "end scan")
	}
	return results, cur.String(), nil
}

// Tally encapsulates statistics about the state of the voting tokens for
// a particular agenda item.
//
// Note:
//
// * Every token must be in _one_ of the distributed, registered or voted
//   states.
//
//   Circulation = Distributed + Registered + Voted
//
// * Every token in the voted state has a vote in the Votes slice.
//
//   Voted = Votes[0] + Votes[1] + ... + Votes[n-1]
//
// * Either none of the voting tokens are closed, or all of the voting tokens
//   are closed.
//
//   Closed = 0 || Closed = Circulation
//
type Tally struct {
	AssetID     bc.AssetID `json:"voting_token_asset_id"`
	Circulation int        `json:"circulation"`
	Distributed int        `json:"distributed"`
	Registered  int        `json:"registered"`
	Voted       int        `json:"voted"`
	Closed      int        `json:"closed"`
	Votes       []int      `json:"votes"`
}

// TallyVotes looks up all voting tokens for the provided asset ID and
// totals the number of tokens in each state and which have voted for
// each possible option.
func TallyVotes(ctx context.Context, tokenAssetID bc.AssetID) (tally Tally, err error) {
	tally.AssetID = tokenAssetID
	const (
		stateQ = `
			SELECT
				option_count,
				SUM(amount) AS total,
				SUM(CASE WHEN state = $1 THEN amount ELSE 0 END) AS distributed,
				SUM(CASE WHEN state = $2 THEN amount ELSE 0 END) AS registered,
				SUM(CASE WHEN state = $3 THEN amount ELSE 0 END) AS voted,
				SUM(CASE WHEN closed THEN amount ELSE 0 END) AS closed
			FROM voting_tokens WHERE asset_id = $4
			GROUP BY option_count
		`
		voteQ = `
			SELECT vote, SUM(amount) AS total
			FROM voting_tokens
			WHERE asset_id = $1 AND state = $2
			GROUP BY vote
		`
	)
	var optionCount int
	err = pg.FromContext(ctx).QueryRow(ctx, stateQ, stateDistributed, stateRegistered, stateVoted, tokenAssetID).
		Scan(&optionCount, &tally.Circulation, &tally.Distributed, &tally.Registered, &tally.Voted, &tally.Closed)
	if err == sql.ErrNoRows {
		return tally, pg.ErrUserInputNotFound
	}
	if err != nil {
		return tally, err
	}

	tally.Votes = make([]int, optionCount)
	err = pg.ForQueryRows(ctx, voteQ, tokenAssetID, stateVoted, func(vote int, total int) error {
		if vote > len(tally.Votes) {
			return fmt.Errorf("vote for option %d exceeds option count %d", vote, optionCount)
		}
		if vote <= 0 {
			return errors.New("voting token in voted state but with nonpositive vote value")
		}

		// Votes are 1-indexed within the contract.
		tally.Votes[vote-1] = total
		return nil
	})
	return tally, err
}

// FindTokenForAsset looks up the current state of the voting token with the
// provided token asset ID and voting right asset ID.
func FindTokenForAsset(ctx context.Context, tokenAssetID, rightAssetID bc.AssetID) (*Token, error) {
	const sqlQ = `
		SELECT
			vt.asset_id,
			vt.right_asset_id,
			vt.tx_hash,
			vt.index,
			vt.state,
			vt.closed,
			vt.vote,
			vt.option_count,
			vt.secret_hash,
			vt.admin_script,
			vt.amount
		FROM voting_tokens vt
		WHERE
			vt.asset_id = $1 AND vt.right_asset_id = $2
	`
	var (
		tok       Token
		baseState int64
		closed    bool
	)
	err := pg.FromContext(ctx).QueryRow(ctx, sqlQ, tokenAssetID, rightAssetID).Scan(
		&tok.AssetID, &tok.Right, &tok.Outpoint.Hash, &tok.Outpoint.Index, &baseState,
		&closed, &tok.Vote, &tok.OptionCount, &tok.SecretHash, &tok.AdminScript, &tok.Amount)
	if err == sql.ErrNoRows {
		return nil, pg.ErrUserInputNotFound
	} else if err != nil {
		return nil, errors.Wrap(err, "looking up voting token by asset id")
	}

	tok.State = TokenState(baseState)
	if closed {
		tok.State = tok.State | stateFinished
	}
	return &tok, nil
}

// GetVotes looks up all of the issued voting tokens with any of the provided
// asset IDs. It supports pagination. Optionally, an account ID can be provided
// to filter the result to votes that have the provided account ID somewhere in
// the vote's active chain of ownership. The account doesn't need to be the
// current holder of the voting right, but it must at least have recall
// privilege over the voting right.
func GetVotes(ctx context.Context, assetIDs []bc.AssetID, accountID string, after string, limit int) ([]*Token, string, error) {
	// Parse the compound cursor
	var cursorToken, cursorRight string
	if a := strings.SplitN(after, "-", 2); len(a) == 2 {
		cursorToken, cursorRight = a[0], a[1]
	}

	// Convert assetIDs to a string slice and filter any asset IDs
	// already eliminated by the cursor.
	var tokenAssetIDs []string
	for _, assetID := range assetIDs {
		if s := assetID.String(); s >= cursorToken {
			tokenAssetIDs = append(tokenAssetIDs, s)
		}
	}

	const (
		qFmt = `
			SELECT
				vt.asset_id,
				vt.right_asset_id,
				vt.tx_hash,
				vt.index,
				vt.state,
				vt.closed,
				vt.vote,
				vt.option_count,
				vt.secret_hash,
				vt.admin_script,
				vt.amount,
				vr.account_id
			FROM voting_tokens vt
			INNER JOIN voting_right_txs vr ON vt.right_asset_id = vr.asset_id AND NOT vr.void
			INNER JOIN utxos u ON (u.tx_hash, u.index) = (vr.tx_hash, vr.index)
			WHERE
				NOT EXISTS (SELECT 1 FROM pool_inputs pi WHERE (pi.tx_hash, pi.index) = (u.tx_hash, u.index))
				AND vt.asset_id = ANY($1) AND (vt.asset_id, vt.right_asset_id) > ($2, $3) %s
			ORDER BY vt.asset_id ASC, vt.right_asset_id ASC
			LIMIT %d
		`
		qAccFilter = `
			AND vt.right_asset_id IN (
				SELECT asset_id FROM voting_right_txs WHERE NOT void AND account_id = $4
			)
		`
	)

	params := []interface{}{pg.Strings(tokenAssetIDs), cursorToken, cursorRight}

	// Include an additional WHERE condition if we're filtering by account ID.
	var additionalWhereQ string
	if accountID != "" {
		additionalWhereQ = qAccFilter
		params = append(params, accountID)
	}

	q := fmt.Sprintf(qFmt, additionalWhereQ, limit)
	rows, err := pg.Query(ctx, q, params...)
	if err != nil {
		return nil, "", errors.Wrap(err, "querying voting tokens")
	}
	defer rows.Close()

	var (
		results []*Token
		last    string
	)
	for rows.Next() {
		var (
			token     Token
			baseState int64
			closed    bool
		)
		err = rows.Scan(
			&token.AssetID, &token.Right, &token.Outpoint.Hash, &token.Outpoint.Index,
			&baseState, &closed, &token.Vote, &token.OptionCount, &token.SecretHash,
			&token.AdminScript, &token.Amount, &token.AccountID,
		)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning Token")
		}
		token.State = TokenState(baseState)
		if closed {
			token.State = token.State | stateFinished
		}
		results = append(results, &token)
		last = fmt.Sprintf("%s-%s", token.AssetID, token.Right)
	}
	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "end scan")
	}

	return results, last, nil
}
