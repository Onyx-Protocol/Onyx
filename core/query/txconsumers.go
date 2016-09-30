package query

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"chain/core/txconsumer"
	"chain/errors"
)

// TxConsumers queries the blockchain for txconsumers matching the query.
func (ind *Indexer) TxConsumers(ctx context.Context, after string, limit int) ([]*txconsumer.TxConsumer, string, error) {
	queryStr, queryArgs := constructTxConsumersQuery(after, limit)
	rows, err := ind.db.Query(ctx, queryStr, queryArgs...)
	if err != nil {
		return nil, "", errors.Wrap(err, "executing txconsumers query")
	}
	defer rows.Close()

	txconsumers := make([]*txconsumer.TxConsumer, 0, limit)
	for rows.Next() {
		var (
			consumer    txconsumer.TxConsumer
			alias       sql.NullString
			isAscending bool
		)
		err := rows.Scan(&consumer.ID, &alias, &consumer.Filter, &consumer.After, &isAscending)
		if err != nil {
			return nil, "", errors.Wrap(err, "scanning txconsumer row")
		}

		if isAscending {
			consumer.Order = "asc"
		} else {
			consumer.Order = "desc"
		}

		if alias.Valid {
			consumer.Alias = &alias.String
		}

		after = consumer.ID
		txconsumers = append(txconsumers, &consumer)
	}
	err = rows.Err()
	if err != nil {
		return nil, "", errors.Wrap(err)
	}

	return txconsumers, after, nil
}

func constructTxConsumersQuery(after string, limit int) (string, []interface{}) {
	var vals []interface{}

	q := "SELECT id, alias, filter, after, is_ascending FROM txconsumers WHERE "
	// add after conditions
	q += fmt.Sprintf("($%d='' OR id < $%d) ", len(vals)+1, len(vals)+1)
	vals = append(vals, after)

	q += "ORDER BY id DESC "
	q += "LIMIT " + strconv.Itoa(limit)
	return q, vals
}
