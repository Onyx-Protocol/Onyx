package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"chain/core/query/filter"
	"chain/errors"
)

// SaveAnnotatedAsset saves an annotated asset to the query indexes.
func (ind *Indexer) SaveAnnotatedAsset(ctx context.Context, asset *AnnotatedAsset, sortID string) error {
	keysJSON, err := json.Marshal(asset.Keys)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `
		INSERT INTO annotated_assets
			(id, sort_id, alias, issuance_program, keys, quorum, definition, tags, local)
		VALUES($1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, $9)
		ON CONFLICT (id) DO UPDATE SET sort_id = $2, tags = $8::jsonb
	`
	_, err = ind.db.ExecContext(ctx, q, asset.ID, sortID, asset.Alias, []byte(asset.IssuanceProgram),
		keysJSON, asset.Quorum, string(*asset.Definition), string(*asset.Tags), bool(asset.IsLocal))
	return errors.Wrap(err, "saving annotated asset")
}

// Assets queries the blockchain for annotated assets matching the query.
func (ind *Indexer) Assets(ctx context.Context, filt string, vals []interface{}, after string, limit int) ([]*AnnotatedAsset, string, error) {
	p, err := filter.Parse(filt, assetsTable, vals)
	if err != nil {
		return nil, "", err
	}
	if len(vals) != p.Parameters {
		return nil, "", ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, assetsTable, vals)
	if err != nil {
		return nil, "", errors.Wrap(err, "converting to SQL")
	}

	queryStr, queryArgs := constructAssetsQuery(expr, vals, after, limit)
	rows, err := ind.db.QueryContext(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing assets query")
	}
	defer rows.Close()

	assets := make([]*AnnotatedAsset, 0, limit)
	for rows.Next() {
		aa := new(AnnotatedAsset)

		var sortID string
		var keysJSON []byte

		err := rows.Scan(
			&aa.ID,
			&sortID,
			&aa.Alias,
			&aa.IssuanceProgram,
			&keysJSON,
			&aa.Quorum,
			&aa.Definition,
			&aa.Tags,
			&aa.IsLocal,
		)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning annotated asset row")
		}
		err = json.Unmarshal(keysJSON, &aa.Keys)
		if err != nil {
			return nil, "", errors.Wrap(err, "unmarshaling asset keys json")
		}

		after = sortID
		assets = append(assets, aa)
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	return assets, after, nil
}

func constructAssetsQuery(expr string, vals []interface{}, after string, limit int) (string, []interface{}) {
	var buf bytes.Buffer

	buf.WriteString("SELECT ")
	buf.WriteString("id, sort_id, alias, issuance_program, keys, quorum, definition, tags, local")
	buf.WriteString(" FROM annotated_assets AS ast")
	buf.WriteString(" WHERE ")

	// add filter conditions
	if len(expr) > 0 {
		buf.WriteString("(")
		buf.WriteString(expr)
		buf.WriteString(") AND ")
	}

	// add after conditions
	buf.WriteString(fmt.Sprintf("($%d='' OR sort_id < $%d) ", len(vals)+1, len(vals)+1))
	vals = append(vals, after)

	buf.WriteString("ORDER BY sort_id DESC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
