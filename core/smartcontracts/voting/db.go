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

// Right encapsulates a claim to a right token.
//
// AssetID   - The asset ID of the voting right token.
// Ordinal   - The position of this owner in the history of this voting right.
//             The ordinal is monotonically increasing with time.
// AccountID - The account id that has a claim to the voting right token. This
//             may be nil if it's an account on another node.
//
type Right struct {
	AssetID   bc.AssetID
	Ordinal   int
	Outpoint  bc.Outpoint
	AccountID *string
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
	prevAssetID string
	prevOrdinal int
}

func (c cursor) String() string {
	if c.prevAssetID == "" && c.prevOrdinal == 0 {
		return ""
	}
	return fmt.Sprintf("%s-%d", c.prevAssetID, c.prevOrdinal)
}

func insertVotingRight(ctx context.Context, assetID bc.AssetID, ordinal int, blockHeight uint64, outpoint bc.Outpoint, data rightScriptData) error {
	const q = `
		INSERT INTO voting_rights
			(asset_id, ordinal, account_id, tx_hash, index, holder, deadline, delegatable, ownership_chain, admin_script, block_height)
			VALUES($1, $2, (SELECT account_id FROM addresses WHERE pk_script=$5), $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (asset_id, ordinal) DO NOTHING
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, assetID, ordinal, outpoint.Hash, outpoint.Index,
		data.HolderScript, data.Deadline, data.Delegatable, data.OwnershipChain[:], data.AdminScript, blockHeight)
	return errors.Wrap(err, "inserting into voting_rights")
}

func upsertVotingToken(ctx context.Context, assetID bc.AssetID, blockHeight uint64, outpoint bc.Outpoint, amount uint64, data tokenScriptData) error {
	const q = `
		INSERT INTO voting_tokens
			(asset_id, right_asset_id, tx_hash, index, state, closed, vote, option_count, secret_hash, admin_script, amount, block_height)
			VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (asset_id, right_asset_id) DO UPDATE
		  SET tx_hash = $3, index = $4, state = $5, closed = $6, vote = $7, secret_hash = $9, block_height = $12
		  WHERE voting_tokens.block_height <= excluded.block_height
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, assetID, data.Right,
		outpoint.Hash, outpoint.Index, data.State.Base(), data.State.Finished(),
		data.Vote, data.OptionCount, data.SecretHash, data.AdminScript, amount, blockHeight)
	return errors.Wrap(err, "upserting into voting_tokens")
}

// voidVotingRights takes an ordinal interval for a voting right asset, and
// voids all voting rights that fall into the interval. Both sides of the
// interval are inclusive.
func voidVotingRights(ctx context.Context, assetID bc.AssetID, blockHeight uint64, startOrdinal, endOrdinal int) error {
	const q = `
		UPDATE voting_rights SET void_block_height = $4
		WHERE asset_id = $1 AND ordinal >= $2 AND ordinal <= $3 AND void_block_height IS NULL
	`
	_, err := pg.FromContext(ctx).Exec(ctx, q, assetID, startOrdinal, endOrdinal, blockHeight)
	return errors.Wrap(err, "voiding voting_rights")
}

type votingRightsQuery struct {
	accountID   string
	outpoint    *bc.Outpoint
	assetID     *bc.AssetID
	includeVoid bool

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
		whereClauses []string
		values       []interface{}
		param        int = 1
	)

	if q.accountID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("vr.account_id = $%d\n", param))
		values = append(values, q.accountID)
		param++
	}
	if q.outpoint != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("vr.tx_hash = $%d AND vr.index = $%d\n", param, param+1))
		values = append(values, q.outpoint.Hash, q.outpoint.Index)
		param += 2
	}
	if q.assetID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("vr.asset_id = $%d\n", param))
		values = append(values, *q.assetID)
		param++
	}
	if q.cursor != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("(vr.asset_id, vr.ordinal) < ($%d, $%d)\n", param, param+1))
		values = append(values, q.cursor.prevAssetID, q.cursor.prevOrdinal)
		param += 2
	}
	if !q.includeVoid {
		whereClauses = append(whereClauses, "vr.void_block_height IS NULL\n")
	}

	if len(whereClauses) == 0 {
		return "", nil
	}
	return "WHERE " + strings.Join(whereClauses, " AND "), values
}

// FindRightsForAccount returns all voting rights belonging to the provided account.
func FindRightsForAccount(ctx context.Context, accountID string, prev string, limit int) ([]*Right, map[bc.AssetID]string, string, error) {
	// Since the sort criteria is composite, the cursor is composite.
	var (
		prevAssetID string
		prevOrdinal int
		cur         cursor
	)
	_, err := fmt.Sscanf(prev, "%s-%d", &prevAssetID, &prevOrdinal)
	if err == nil { // ignore malformed cursors
		cur = cursor{
			prevAssetID: prevAssetID,
			prevOrdinal: prevOrdinal,
		}
	}

	rights, next, err := findVotingRights(ctx, votingRightsQuery{
		accountID: accountID,
		cursor:    &cur,
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
		FROM voting_rights vr
		INNER JOIN (
			SELECT asset_id, MAX(ordinal) AS ordinal
			FROM voting_rights
			WHERE void_block_height IS NULL
			GROUP BY asset_id
		) vr_utxos
		ON (vr.asset_id, vr.ordinal) = (vr_utxos.asset_id, vr_utxos.ordinal)
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

// FindRightsForAsset return all non-void claims to the voting right
// token with the provided asset ID. The resulting voting rights will
// be sorted chronologically (by ordinal).
func FindRightsForAsset(ctx context.Context, assetID bc.AssetID) ([]*Right, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{assetID: &assetID})
	if err != nil {
		return nil, err
	}
	return rights, nil
}

// FindRightPrevout looks up all voting rights created at the provided
// outpoint, and returns the one with the highest ordinal. This right
// will contain the current holder at that outpoint and the ownership
// chain at this outpoint.
//
// This function may return a voided voting right if `outpoint` is not
// the current utxo for the voting right asset.
func FindRightPrevout(ctx context.Context, assetID bc.AssetID, outpoint bc.Outpoint) (*Right, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{
		assetID:     &assetID,
		outpoint:    &outpoint,
		includeVoid: true,
	})
	if err != nil {
		return nil, err
	}
	if len(rights) == 0 {
		return nil, pg.ErrUserInputNotFound
	}
	return rights[len(rights)-1], nil
}

// GetCurrentHolder looks up the current utxo for the voting right with the
// provided assetID.
func GetCurrentHolder(ctx context.Context, assetID bc.AssetID) (*Right, error) {
	rights, _, err := findVotingRights(ctx, votingRightsQuery{assetID: &assetID})
	if err != nil {
		return nil, err
	}
	if len(rights) == 0 {
		return nil, pg.ErrUserInputNotFound
	}
	return rights[len(rights)-1], nil
}

// findRecallOrdinal looks up the ordinal of a recall point. It's used during
// voting right indexing and will look up the largest ordinal still less than
// the previous outpoint's ordinal that has a matching ownership chain.
func findRecallOrdinal(ctx context.Context, assetID bc.AssetID, prevoutOrdinal int, recallChain bc.Hash) (recallOrdinal int, err error) {
	const sqlQ = `
		SELECT ordinal FROM voting_rights vr
		WHERE asset_id = $1 AND ordinal < $2 AND ownership_chain = $3
		ORDER BY ordinal DESC LIMIT 1
	`
	err = pg.QueryRow(ctx, sqlQ, assetID, prevoutOrdinal, recallChain[:]).Scan(&recallOrdinal)
	if err == sql.ErrNoRows {
		return 0, pg.ErrUserInputNotFound
	}
	return recallOrdinal, err
}

func findVotingRights(ctx context.Context, q votingRightsQuery) ([]*Right, string, error) {
	var (
		cur     cursor
		results []*Right
	)

	const sqlQ = `
		SELECT
			vr.asset_id,
			vr.ordinal,
			vr.tx_hash,
			vr.index,
			vr.account_id,
			vr.holder,
			vr.deadline,
			vr.delegatable,
			vr.ownership_chain,
			vr.admin_script
		FROM voting_rights vr
	`
	whereSQL, values := q.Where()
	queryStr := fmt.Sprintf("%s%s ORDER BY vr.asset_id ASC, vr.ordinal ASC%s", sqlQ, whereSQL, q.Limit())
	rows, err := pg.Query(ctx, queryStr, values...)
	if err != nil {
		return nil, "", errors.Wrap(err, "querying findVotingRights")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			right          Right
			ownershipChain []byte
		)

		err = rows.Scan(
			&right.AssetID, &right.Ordinal,
			&right.Outpoint.Hash, &right.Outpoint.Index,
			&right.AccountID, &right.HolderScript, &right.Deadline,
			&right.Delegatable, &ownershipChain, &right.AdminScript)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning Right")
		}
		copy(right.OwnershipChain[:], ownershipChain)
		results = append(results, &right)
		cur = cursor{
			prevAssetID: right.AssetID.String(),
			prevOrdinal: right.Ordinal,
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
		if vote >= len(tally.Votes) {
			return fmt.Errorf("vote for option %d exceeds option count %d", vote, optionCount)
		}
		if vote < 0 {
			return errors.New("voting token in voted state but with negative vote value")
		}

		// Votes are 1-indexed within the contract.
		tally.Votes[vote] = total
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
			INNER JOIN voting_rights vr ON vt.right_asset_id = vr.asset_id AND vr.void_block_height IS NULL
			INNER JOIN (
				SELECT asset_id, MAX(ordinal) AS ordinal
				FROM voting_rights
				WHERE void_block_height IS NULL
				GROUP BY asset_id
			) vr_utxos
			ON (vr.asset_id, vr.ordinal) = (vr_utxos.asset_id, vr_utxos.ordinal)
			WHERE
				vt.asset_id = ANY($1) AND (vt.asset_id, vt.right_asset_id) > ($2, $3) %s
			ORDER BY vt.asset_id ASC, vt.right_asset_id ASC
			LIMIT %d
		`
		qAccFilter = `
			AND vt.right_asset_id IN (
				SELECT asset_id FROM voting_rights WHERE void_block_height IS NULL AND account_id = $4
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
