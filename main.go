package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const (
	dbHost     = "localhost"
	dbPort     = 5432
	dbUser     = "postgres"
	dbPassword = "postgres"
	dbName     = "audience_db"
)

func connectDB() (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// Old EAV model - slow query
func oldEAVQuery(db *sql.DB, audienceRule string) (int, time.Duration, error) {
	query := `
		SELECT COUNT(DISTINCT u.user_id)
		FROM users u
		WHERE EXISTS (
			SELECT 1 FROM user_attributes ua
			WHERE ua.user_id = u.user_id
			AND ua.key = 'country'
			AND ua.value = 'US'
		)`

	start := time.Now()
	var count int
	err := db.QueryRow(query).Scan(&count)
	duration := time.Since(start)

	return count, duration, err
}

// Complex EAV query
func oldEAVComplexQuery(db *sql.DB) (int, time.Duration, error) {
	query := `
		SELECT COUNT(DISTINCT u.user_id)
		FROM users u
		WHERE EXISTS (
			SELECT 1 FROM user_attributes ua1
			WHERE ua1.user_id = u.user_id
			AND ua1.key = 'country'
			AND ua1.value = 'US'
		)
		OR EXISTS (
			SELECT 1 FROM user_attributes ua2
			WHERE ua2.user_id = u.user_id
			AND ua2.key = 'tier'
			AND ua2.value IN ('gold', 'platinum')
		)`

	start := time.Now()
	var count int
	err := db.QueryRow(query).Scan(&count)
	duration := time.Since(start)

	return count, duration, err
}

// New optimized model - fast query
func optimizedQuery(db *sql.DB, audienceRule string) (int, time.Duration, error) {
	query := `
		SELECT COUNT(*)
		FROM user_profiles
		WHERE country = 'US'`

	start := time.Now()
	var count int
	err := db.QueryRow(query).Scan(&count)
	duration := time.Since(start)

	return count, duration, err
}

// Complex optimized query
func optimizedComplexQuery(db *sql.DB) (int, time.Duration, error) {
	query := `
		SELECT COUNT(*)
		FROM user_profiles
		WHERE country = 'US'
		   OR tier IN ('gold', 'platinum')`

	start := time.Now()
	var count int
	err := db.QueryRow(query).Scan(&count)
	duration := time.Since(start)

	return count, duration, err
}

// AND query for optimized model
func optimizedANDQuery(db *sql.DB) (int, time.Duration, error) {
	query := `
		SELECT COUNT(*)
		FROM user_profiles
		WHERE has_purchased = true
		  AND total_spend > 100`

	start := time.Now()
	var count int
	err := db.QueryRow(query).Scan(&count)
	duration := time.Since(start)

	return count, duration, err
}

// Show EXPLAIN ANALYZE for query
func explainQuery(db *sql.DB, query string) {
	explainQuery := "EXPLAIN ANALYZE " + query
	rows, err := db.Query(explainQuery)
	if err != nil {
		log.Printf("Error explaining query: %v", err)
		return
	}
	defer rows.Close()

	fmt.Println("\n📊 Query Plan:")
	for rows.Next() {
		var plan string
		if err := rows.Scan(&plan); err != nil {
			continue
		}
		fmt.Println("  ", plan)
	}
}

func main() {
	fmt.Println("🚀 Audience Service Performance Test with Real PostgreSQL")
	fmt.Println(strings.Repeat("=", 60))

	db, err := connectDB()
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Database is not responding:", err)
	}

	fmt.Println("✅ Connected to PostgreSQL")

	var userCount int
	db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	fmt.Printf("\n📈 Test dataset: %d users\n\n", userCount)

	fmt.Println("📊 Test 1: Simple Query (country = 'US')")
	fmt.Println(strings.Repeat("-", 50))

	count1, duration1, err := oldEAVQuery(db, "country = 'US'")
	if err != nil {
		log.Printf("EAV query error: %v", err)
	} else {
		fmt.Printf("EAV Model:        %6d users in %v\n", count1, duration1)
	}

	count2, duration2, err := optimizedQuery(db, "country = 'US'")
	if err != nil {
		log.Printf("Optimized query error: %v", err)
	} else {
		fmt.Printf("Optimized Model:  %6d users in %v\n", count2, duration2)
	}

	if duration1 > 0 && duration2 > 0 {
		speedup := float64(duration1) / float64(duration2)
		fmt.Printf("⚡ Speedup:        %.1fx\n", speedup)
	}

	fmt.Println("\n📊 Test 2: Complex OR Query")
	fmt.Println(strings.Repeat("-", 50))

	count3, duration3, err := oldEAVComplexQuery(db)
	if err != nil {
		log.Printf("Complex EAV query error: %v", err)
	} else {
		fmt.Printf("EAV Model:        %6d users in %v\n", count3, duration3)
	}

	count4, duration4, err := optimizedComplexQuery(db)
	if err != nil {
		log.Printf("Complex optimized query error: %v", err)
	} else {
		fmt.Printf("Optimized Model:  %6d users in %v\n", count4, duration4)
	}

	if duration3 > 0 && duration4 > 0 {
		speedup := float64(duration3) / float64(duration4)
		fmt.Printf("⚡ Speedup:        %.1fx\n", speedup)
	}

	fmt.Println("\n📊 Test 3: Complex AND Query")
	fmt.Println(strings.Repeat("-", 50))

	count5, duration5, err := optimizedANDQuery(db)
	if err != nil {
		log.Printf("AND query error: %v", err)
	} else {
		fmt.Printf("Optimized Model:  %6d users in %v\n", count5, duration5)
	}

	fmt.Println("\n🔍 Query Execution Plan (Optimized Model):")
	explainQuery(db, "SELECT COUNT(*) FROM user_profiles WHERE country = 'US'")

	fmt.Println("\n📈 Summary:")
	fmt.Println(strings.Repeat("-", 50))
	fmt.Printf("Dataset size:     %d users\n", userCount)
	if duration1 > 0 && duration2 > 0 {
		avgSpeedup := float64(duration1+duration3) / float64(duration2+duration4)
		fmt.Printf("Average speedup:  %.1fx\n", avgSpeedup)
		fmt.Printf("Target achieved:  %v\n", duration2 < 2*time.Second && duration4 < 2*time.Second)
	}

	// Extrapolation to 10M users
	if userCount < 10000000 && duration2 > 0 {
		scaleFactor := float64(10000000) / float64(userCount)
		estimatedTime := time.Duration(float64(duration2) * scaleFactor)
		fmt.Printf("\n🔮 Estimated for 10M users: %v\n", estimatedTime)
		fmt.Printf("   Target <2s:     %v\n", estimatedTime < 2*time.Second)
	}
}
