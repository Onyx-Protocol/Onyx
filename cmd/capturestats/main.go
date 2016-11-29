package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

func main() {
	captureDBStats(os.Getenv("DATABASE_URL"))
}

func captureDBStats(dburl string) {
	db, err := sql.Open("postgres", dburl)
	if err != nil {
		log.Printf("error capturing db stats: %s", err.Error())
		return
	}

	const q = `
		SELECT
			(total_time / 1000 / 60) as total_minutes,
			(total_time/calls) as average_time,
			calls,
			query
		FROM pg_stat_statements
		ORDER BY 1 DESC
		LIMIT 100;
	`
	rows, err := db.Query(q)
	if err != nil {
		log.Printf("error capturing db stats: %s", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var (
			totalMin, avgTimeMS float64
			ncalls              uint64
			query               string
		)
		err := rows.Scan(&totalMin, &avgTimeMS, &ncalls, &query)
		if err != nil {
			log.Printf("error capturing db stats: %s", err.Error())
			return
		}
		fmt.Printf(
			"Total Minutes: %f\nAverage MS: %f\nCalls: %d\nQuery: %s\n---\n",
			totalMin,
			avgTimeMS,
			ncalls,
			query,
		)
	}
	if err := rows.Err(); err != nil {
		log.Printf("error capturing db stats: %s", err.Error())
	}
}
