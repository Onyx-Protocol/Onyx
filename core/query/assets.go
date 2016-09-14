package query

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/lib/pq"

	"chain/core/query/filter"
	"chain/core/signers"
	"chain/errors"
	"chain/protocol/bc"
)

// SaveAnnotatedAsset saves an annotated asset to the query indexes.
func (ind *Indexer) SaveAnnotatedAsset(ctx context.Context, assetID bc.AssetID, asset map[string]interface{}, sortID string) error {
	b, err := json.Marshal(asset)
	if err != nil {
		return errors.Wrap(err)
	}

	const q = `
		INSERT INTO annotated_assets (id, data, sort_id) VALUES($1, $2, $3)
		ON CONFLICT (id) DO UPDATE SET data = $2, sort_id = $3
	`
	_, err = ind.db.Exec(ctx, q, assetID.String(), b, sortID)
	return errors.Wrap(err, "saving annotated asset")
}

// Assets queries the blockchain for annotated assets matching the query.
func (ind *Indexer) Assets(ctx context.Context, p filter.Predicate, vals []interface{}, after string, limit int) ([]map[string]interface{}, string, error) {
	if len(vals) != p.Parameters {
		return nil, "", ErrParameterCountMismatch
	}
	expr, err := filter.AsSQL(p, "data", vals)
	if err != nil {
		return nil, "", errors.Wrap(err, "converting to SQL")
	}

	queryStr, queryArgs := constructAssetsQuery(expr, after, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing assets query")
	}
	defer rows.Close()

	var (
		assets     = make([]map[string]interface{}, 0, limit)
		assetsByID = make(map[string]map[string]interface{})
		ids        []string
	)
	for rows.Next() {
		var (
			id, sortID string
			rawAsset   []byte
		)
		err := rows.Scan(&id, &sortID, &rawAsset)
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

		after = sortID
		assets = append(assets, asset)
		ids = append(ids, id)
		assetsByID[id] = asset
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	// Attach signer information (xpubs, quorum) to local assets
	const q = `
		SELECT assets.id, signers.xpubs, signers.quorum
		FROM assets
		JOIN signers ON assets.signer_id = signers.id
		WHERE assets.id IN (SELECT unnest($1::text[]))
	`
	rows, err = ind.db.Query(ctx, q, pq.StringArray(ids))
	if err != nil {
		return nil, "", errors.Wrap(err, "signers query")
	}
	defer rows.Close()

	for rows.Next() {
		var (
			id       string
			rawXpubs pq.StringArray
			quorum   int
		)

		err := rows.Scan(&id, &rawXpubs, &quorum)
		if err != nil {
			return nil, "", errors.Wrap(err, "signers scan")
		}

		xpubs, err := signers.ConvertKeys(rawXpubs)
		if err != nil { // silently ignore errors
			asset := assetsByID[id]
			asset["xpubs"] = xpubs
			asset["quorum"] = quorum
		}
	}
	if err = rows.Err(); err != nil {
		return nil, "", errors.Wrap(err, "signers end row scan loop")
	}

	return assets, after, nil
}

func constructAssetsQuery(expr filter.SQLExpr, after string, limit int) (string, []interface{}) {
	var buf bytes.Buffer
	var vals []interface{}

	buf.WriteString("SELECT id, sort_id, data FROM annotated_assets")
	buf.WriteString(" WHERE ")

	// add filter conditions
	if len(expr.Values) > 0 {
		vals = append(vals, expr.Values...)
		buf.WriteString("(")
		buf.WriteString(expr.SQL)
		buf.WriteString(") AND ")
	}

	// add after conditions
	buf.WriteString(fmt.Sprintf("($%d='' OR sort_id < $%d) ", len(vals)+1, len(vals)+1))
	vals = append(vals, string(after))

	buf.WriteString("ORDER BY sort_id DESC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
