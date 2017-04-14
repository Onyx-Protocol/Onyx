package query

import (
	"chain/core/query/filter"
)

var (
	assetsTable = &filter.SQLTable{
		Name:  "annotated_assets",
		Alias: "ast",
		Columns: map[string]*filter.SQLColumn{
			"id":               {Name: "id", Type: filter.String, SQLType: filter.SQLBytea},
			"alias":            {Name: "alias", Type: filter.String, SQLType: filter.SQLText},
			"issuance_program": {Name: "issuance_program", Type: filter.String, SQLType: filter.SQLBytea},
			"quorum":           {Name: "quorum", Type: filter.Integer, SQLType: filter.SQLInteger},
			"tags":             {Name: "tags", Type: filter.Object, SQLType: filter.SQLJSONB},
			"definition":       {Name: "definition", Type: filter.Object, SQLType: filter.SQLJSONB},
			"is_local":         {Name: "local", Type: filter.String, SQLType: filter.SQLBool},
		},
	}
	accountsTable = &filter.SQLTable{
		Name:  "annotated_accounts",
		Alias: "acc",
		Columns: map[string]*filter.SQLColumn{
			"id":     {Name: "id", Type: filter.String, SQLType: filter.SQLText},
			"alias":  {Name: "alias", Type: filter.String, SQLType: filter.SQLText},
			"quorum": {Name: "quorum", Type: filter.Integer, SQLType: filter.SQLInteger},
			"tags":   {Name: "tags", Type: filter.Object, SQLType: filter.SQLJSONB},
		},
	}
	outputsTable = &filter.SQLTable{
		Name:  "annotated_outputs",
		Alias: "out",
		Columns: map[string]*filter.SQLColumn{
			"id":               {Name: "output_id", Type: filter.String, SQLType: filter.SQLBytea},
			"type":             {Name: "type", Type: filter.String, SQLType: filter.SQLText},
			"purpose":          {Name: "purpose", Type: filter.String, SQLType: filter.SQLText},
			"transaction_id":   {Name: "tx_hash", Type: filter.String, SQLType: filter.SQLBytea},
			"position":         {Name: "output_index", Type: filter.Integer, SQLType: filter.SQLInteger},
			"asset_id":         {Name: "asset_id", Type: filter.String, SQLType: filter.SQLBytea},
			"asset_alias":      {Name: "asset_alias", Type: filter.String, SQLType: filter.SQLText},
			"asset_definition": {Name: "asset_definition", Type: filter.Object, SQLType: filter.SQLJSONB},
			"asset_tags":       {Name: "asset_tags", Type: filter.Object, SQLType: filter.SQLJSONB},
			"asset_is_local":   {Name: "asset_local", Type: filter.String, SQLType: filter.SQLBool},
			"amount":           {Name: "amount", Type: filter.Integer, SQLType: filter.SQLBigint},
			"account_id":       {Name: "account_id", Type: filter.String, SQLType: filter.SQLText},
			"account_alias":    {Name: "account_alias", Type: filter.String, SQLType: filter.SQLText},
			"account_tags":     {Name: "account_tags", Type: filter.Object, SQLType: filter.SQLJSONB},
			"control_program":  {Name: "control_program", Type: filter.String, SQLType: filter.SQLBytea},
			"reference_data":   {Name: "reference_data", Type: filter.Object, SQLType: filter.SQLJSONB},
			"is_local":         {Name: "local", Type: filter.String, SQLType: filter.SQLBool},
		},
	}
	inputsTable = &filter.SQLTable{
		Name:  "annotated_inputs",
		Alias: "inp",
		Columns: map[string]*filter.SQLColumn{
			"type":             {Name: "type", Type: filter.String, SQLType: filter.SQLText},
			"asset_id":         {Name: "asset_id", Type: filter.String, SQLType: filter.SQLBytea},
			"asset_alias":      {Name: "asset_alias", Type: filter.String, SQLType: filter.SQLText},
			"asset_definition": {Name: "asset_definition", Type: filter.Object, SQLType: filter.SQLJSONB},
			"asset_tags":       {Name: "asset_tags", Type: filter.Object, SQLType: filter.SQLJSONB},
			"asset_is_local":   {Name: "asset_local", Type: filter.String, SQLType: filter.SQLBool},
			"amount":           {Name: "amount", Type: filter.Integer, SQLType: filter.SQLBigint},
			"account_id":       {Name: "account_id", Type: filter.String, SQLType: filter.SQLText},
			"account_alias":    {Name: "account_alias", Type: filter.String, SQLType: filter.SQLText},
			"account_tags":     {Name: "account_tags", Type: filter.Object, SQLType: filter.SQLJSONB},
			"issuance_program": {Name: "issuance_program", Type: filter.String, SQLType: filter.SQLBytea},
			"reference_data":   {Name: "reference_data", Type: filter.Object, SQLType: filter.SQLJSONB},
			"is_local":         {Name: "local", Type: filter.String, SQLType: filter.SQLBool},
			"spent_output_id":  {Name: "spent_output_id", Type: filter.String, SQLType: filter.SQLBytea},
			"spent_output":     {Name: "spent_output", Type: filter.Object, SQLType: filter.SQLJSONB},
		},
	}
	transactionsTable = &filter.SQLTable{
		Name:  "annotated_txs",
		Alias: "txs",
		Columns: map[string]*filter.SQLColumn{
			"id":                       {Name: "tx_hash", Type: filter.String, SQLType: filter.SQLBytea},
			"timestamp":                {Name: "timestamp", Type: filter.String, SQLType: filter.SQLTimestamp},
			"block_id":                 {Name: "block_id", Type: filter.String, SQLType: filter.SQLBytea},
			"block_height":             {Name: "block_height", Type: filter.Integer, SQLType: filter.SQLBigint},
			"position":                 {Name: "tx_pos", Type: filter.Integer, SQLType: filter.SQLInteger},
			"block_transactions_count": {Name: "block_tx_count", Type: filter.Integer, SQLType: filter.SQLInteger},
			"reference_data":           {Name: "reference_data", Type: filter.Object, SQLType: filter.SQLJSONB},
			"is_local":                 {Name: "local", Type: filter.String, SQLType: filter.SQLBool},
		},
		ForeignKeys: map[string]*filter.SQLForeignKey{
			"inputs":  {Table: inputsTable, LocalColumn: "tx_hash", ForeignColumn: "tx_hash"},
			"outputs": {Table: outputsTable, LocalColumn: "tx_hash", ForeignColumn: "tx_hash"},
		},
	}
)
