package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"

	"golang.org/x/net/context"

	"chain/core/query/chql"
	"chain/errors"
)

// AccountTags queries the blockchain for accounts with tags matching
// the query `q`. It returns the IDs and a map from the account ID to the
// matching account tags.
func (i *Indexer) AccountTags(ctx context.Context, q chql.Query, vals []interface{}, cur string, limit int) ([]string, map[string]map[string]interface{}, string, error) {
	expr, err := chql.AsSQL(q, "tags", vals)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "converting to SQL")
	}
	if len(expr.GroupBy) > 0 {
		// A GROUP BY query doesn't make sense for accounts. This
		// is caused by leaving a parameter unconstrained in the query.
		return nil, nil, "", ErrMissingParameters
	}

	queryStr, queryArgs := constructAccountsQuery(expr, cur, limit)
	rows, err := i.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "executing acc query")
	}
	defer rows.Close()

	accIDs := make([]string, 0, limit)
	accs := map[string]map[string]interface{}{}
	for rows.Next() {
		var accID string
		var rawTags []byte
		err := rows.Scan(&accID, &rawTags)
		if err != nil {
			return nil, nil, "", errors.Wrap(err, "scanning account tags row")
		}

		var tags map[string]interface{}
		if len(rawTags) > 0 {
			err = json.Unmarshal(rawTags, &tags)
			if err != nil {
				return nil, nil, "", err
			}
		}

		cur = accID
		accIDs = append(accIDs, accID)
		accs[accID] = tags
	}
	err = rows.Err()
	if err != nil {
		return nil, nil, "", errors.Wrap(err)
	}
	return accIDs, accs, cur, nil
}

func constructAccountsQuery(expr chql.SQLExpr, cur string, limit int) (string, []interface{}) {
	var buf bytes.Buffer
	var vals []interface{}

	buf.WriteString("SELECT account_id, tags FROM account_tags")
	buf.WriteString(" WHERE ")

	// add filter conditions
	vals = append(vals, expr.Values...)
	buf.WriteString("(")
	buf.WriteString(expr.SQL)
	buf.WriteString(") AND ")

	// add cursor conditions
	buf.WriteString(fmt.Sprintf("account_id > $%d ", len(vals)+1))
	vals = append(vals, string(cur))

	buf.WriteString("ORDER BY account_id ASC ")
	buf.WriteString("LIMIT " + strconv.Itoa(limit))
	return buf.String(), vals
}
