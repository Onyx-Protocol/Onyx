package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"chain/core/query/chql"
	"chain/cos/bc"
	"chain/errors"
)

// SaveAnnotatedAsset saves an annotated asset to the query indexes.
func (i *Indexer) SaveAnnotatedAsset(ctx context.Context, assetID bc.AssetID, asset map[string]interface{}) error {
	b, err := json.Marshal(asset)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `
		INSERT INTO annotated_assets (id, data) VALUES($1, $2)
		ON CONFLICT (id) DO UPDATE SET data = $2
	`
	_, err = i.db.Exec(ctx, q, assetID.String(), b)
	return errors.Wrap(err, "saving annotated asset")
}

// Assets queries the blockchain for annotated assets matching the query.
func (i *Indexer) Assets(ctx context.Context, q chql.Query, vals []interface{}, cur string, limit int) ([]map[string]interface{}, string, error) {
	if len(vals) != q.Parameters {
		return nil, "", ErrParameterCountMismatch
	}
	expr, err := chql.AsSQL(q, "data", vals)
	if err != nil {
		return nil, "", errors.Wrap(err, "converting to SQL")
	}

	queryStr, queryArgs := constructAssetsQuery(expr, cur, limit)
	rows, err := i.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing assets query")
	}
	defer rows.Close()

	assets := make([]map[string]interface{}, 0, limit)
	for rows.Next() {
		var assetID bc.AssetID
		var rawAsset []byte
		err := rows.Scan(&assetID, &rawAsset)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning annotated asset row")
		}

		var asset map[string]interface{}
		if len(rawAsset) > 0 {
			err = json.Unmarshal(rawAsset, &asset)
			if err != nil {
				return nil, "", err
			}
		}

		cur = assetID.String()
		assets = append(assets, asset)
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}
	return assets, cur, nil
}

func constructAssetsQuery(expr chql.SQLExpr, cur string, limit int) (string, []interface{}) {
	var buf bytes.Buffer
	var vals []interface{}

	buf.WriteString("SELECT id, data FROM annotated_assets")
	buf.WriteString(" WHERE ")

	// add filter conditions
	if len(expr.Values) > 0 {
		vals = append(vals, expr.Values...)
		buf.WriteString("(")
		buf.WriteString(expr.SQL)
		buf.WriteString(") AND ")
	}

	// add cursor conditions
	buf.WriteString(fmt.Sprintf("id > $%d ", len(vals)+1))
	vals = append(vals, string(cur))

	buf.WriteString("ORDER BY id ASC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
