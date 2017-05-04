package query

import (
	"bytes"
	"context"
	"fmt"
	"math"

	"github.com/lib/pq"

	"chain/core/query/filter"
	"chain/errors"
	"chain/protocol/bc"
)

var defaultOutputsAfter = OutputsAfter{
	lastBlockHeight: math.MaxInt64,
	lastTxPos:       math.MaxUint32,
	lastIndex:       math.MaxInt32,
}

type OutputsAfter struct {
	lastBlockHeight uint64
	lastTxPos       uint32
	lastIndex       int
}

func (cur OutputsAfter) String() string {
	return fmt.Sprintf("%d:%d:%d", cur.lastBlockHeight, cur.lastTxPos, cur.lastIndex)
}

func DecodeOutputsAfter(str string) (c *OutputsAfter, err error) {
	var lastBlockHeight, lastTxPos, lastIndex uint64
	_, err = fmt.Sscanf(str, "%d:%d:%d", &lastBlockHeight, &lastTxPos, &lastIndex)
	if err != nil {
		return c, errors.Sub(ErrBadAfter, err)
	}
	if lastBlockHeight > math.MaxInt64 ||
		lastTxPos > math.MaxUint32 ||
		lastIndex > math.MaxInt32 {
		return nil, errors.Wrap(ErrBadAfter)
	}
	return &OutputsAfter{
		lastBlockHeight: lastBlockHeight,
		lastTxPos:       uint32(lastTxPos),
		lastIndex:       int(lastIndex),
	}, nil
}

func (ind *Indexer) Outputs(ctx context.Context, filt string, vals []interface{}, timestampMS uint64, after *OutputsAfter, limit int) ([]*AnnotatedOutput, *OutputsAfter, error) {
	p, err := filter.Parse(filt, outputsTable, vals)
	if err != nil {
		return nil, nil, err
	}
	if len(vals) != p.Parameters {
		return nil, nil, ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, outputsTable, vals)
	if err != nil {
		return nil, nil, err
	}
	queryStr, queryArgs := constructOutputsQuery(expr, vals, timestampMS, after, limit)
	rows, err := ind.db.QueryContext(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var newAfter = defaultOutputsAfter
	if after != nil {
		newAfter = *after
	}

	outputs := make([]*AnnotatedOutput, 0, limit)
	for rows.Next() {
		var (
			blockHeight  uint64
			txPos        uint32
			txID         = new(bc.Hash)
			accountID    *string
			accountAlias *string
			out          = new(AnnotatedOutput)
		)
		err = rows.Scan(
			&blockHeight,
			&txPos,
			&out.Position,
			txID,
			&out.OutputID,
			&out.Type,
			&out.Purpose,
			&out.AssetID,
			&out.AssetAlias,
			&out.AssetDefinition,
			&out.AssetTags,
			&out.AssetIsLocal,
			&out.Amount,
			&accountID,
			&accountAlias,
			&out.AccountTags,
			&out.ControlProgram,
			&out.ReferenceData,
			&out.IsLocal,
		)
		if err != nil {
			return nil, nil, errors.Wrap(err, "scanning annotated output")
		}

		out.TransactionID = txID

		// Set nullable fields.
		if accountID != nil {
			out.AccountID = *accountID
		}
		if accountAlias != nil {
			out.AccountAlias = *accountAlias
		}

		outputs = append(outputs, out)

		newAfter.lastBlockHeight = blockHeight
		newAfter.lastTxPos = txPos
		newAfter.lastIndex = out.Position
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, err
	}

	return outputs, &newAfter, nil
}

func constructOutputsQuery(where string, vals []interface{}, timestampMS uint64, after *OutputsAfter, limit int) (string, []interface{}) {
	var buf bytes.Buffer

	buf.WriteString("SELECT ")
	buf.WriteString("block_height, tx_pos, output_index, tx_hash, output_id, type, purpose, ")
	buf.WriteString("asset_id, asset_alias, asset_definition, asset_tags, asset_local, ")
	buf.WriteString("amount, account_id, account_alias, account_tags, control_program, ")
	buf.WriteString("reference_data, local")
	buf.WriteString(" FROM ")
	buf.WriteString(pq.QuoteIdentifier("annotated_outputs"))
	buf.WriteString(" AS out WHERE ")

	if where != "" {
		buf.WriteString("(")
		buf.WriteString(where)
		buf.WriteString(") AND ")
	}

	vals = append(vals, timestampMS)
	timestampValIndex := len(vals)
	buf.WriteString(fmt.Sprintf("timespan @> $%d::int8", timestampValIndex))

	if after != nil {
		vals = append(vals, after.lastBlockHeight)
		lastBlockHeightValIndex := len(vals)

		vals = append(vals, after.lastTxPos)
		lastTxPosValIndex := len(vals)

		vals = append(vals, after.lastIndex)
		lastIndexValIndex := len(vals)

		buf.WriteString(fmt.Sprintf(" AND (block_height, tx_pos, output_index) < ($%d, $%d, $%d)", lastBlockHeightValIndex, lastTxPosValIndex, lastIndexValIndex))
	}

	buf.WriteString(fmt.Sprintf(" ORDER BY block_height DESC, tx_pos DESC, output_index DESC LIMIT %d", limit))

	return buf.String(), vals
}
