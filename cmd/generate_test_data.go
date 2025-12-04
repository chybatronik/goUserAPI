package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	// Подключение к БД
	db, err := sql.Open("pgx", "host=localhost user=postgres dbname=go_user_api_dev port=5432 sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// Генерация 10,000 записей
	ctx := context.Background()
	stmt, err := db.PrepareContext(ctx, `
          INSERT INTO users (first_name, last_name, age, recording_date)
          VALUES ($1, $2, $3, $4)
      `)
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	firstNames := []string{"Alex", "Maria", "John", "Sarah", "Mike", "Emma",
		"David", "Lisa"}
	lastNames := []string{"Smith", "Johnson", "Brown", "Davis", "Wilson",
		"Miller", "Taylor", "Anderson"}

	fmt.Println("Generating 10,000 test records...")

	for i := 1; i <= 10000; i++ {
		firstName := firstNames[rand.Intn(len(firstNames))]
		lastName := lastNames[rand.Intn(len(lastNames))]
		age := rand.Intn(62) + 18 // 18-80
		recordingDate := time.Now().Add(-time.Duration(rand.Intn(730*24)) *
			time.Hour).Unix()

		_, err := stmt.ExecContext(ctx, firstName, lastName, age, recordingDate)
		if err != nil {
			log.Printf("Error inserting record %d: %v", i, err)
		}

		if i%1000 == 0 {
			fmt.Printf("Generated %d records...\n", i)
		}
	}

	fmt.Println("Test data generation completed!")
}
