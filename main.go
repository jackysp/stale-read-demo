package main

import (
    "database/sql"
    "flag"
    "fmt"
    "log"
    "math/rand"
    "sync"
    "sync/atomic"
    "time"

    _ "github.com/go-sql-driver/mysql"
)

func main() {
    // parse CLI flags
    staleness := flag.Int("staleness", 0, "tidb_read_staleness in seconds, negative for stale reads")
    flag.Parse()

    // TiDB server addresses
    servers := []string{"10.148.0.15:4000", "10.148.0.16:4000", "10.148.0.17:4000"}
    // DSN template connecting to the test database
    dsnTemplate := "root@tcp(%s)/test?charset=utf8mb4&parseTime=True&loc=Local"

    // Connect to the first server to initialize schema
    initDSN := fmt.Sprintf(dsnTemplate, servers[0])
    db, err := sql.Open("mysql", initDSN)
    if err != nil {
        log.Fatalf("failed to open connection: %v", err)
    }
    // defer db.Close() // will close explicitly after data insertion

    createTable := `
    CREATE TABLE IF NOT EXISTS users (
        id INT PRIMARY KEY AUTO_INCREMENT,
        name VARCHAR(64),
        age INT
    ) ENGINE=InnoDB;
    `
    if _, err := db.Exec(createTable); err != nil {
        log.Fatalf("failed to create table: %v", err)
    }

    // Check existing row count and seed data if needed
    var count int
    if err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count); err != nil {
        log.Fatalf("failed to query existing row count: %v", err)
    }
    if count != 10000 {
        // Clear existing rows
        if _, err := db.Exec("TRUNCATE TABLE users"); err != nil {
            log.Fatalf("failed to truncate table: %v", err)
        }
        // Insert sample data
        log.Println("Inserting sample data...")
        stmt, err := db.Prepare("INSERT INTO users(id, name, age) VALUES(?, ?, ?)")
        if err != nil {
            log.Fatalf("failed to prepare insert statement: %v", err)
        }
        for i := 0; i < 10000; i++ {
            id := i + 1
            name := fmt.Sprintf("user_%d", i)
            age := rand.Intn(100)
            if _, err := stmt.Exec(id, name, age); err != nil {
                log.Fatalf("failed to insert row: %v", err)
            }
        }
        stmt.Close()
        log.Println("Data insertion complete.")
    } else {
        log.Println("10000 rows exist, skipping data load")
    }

    if err := db.Close(); err != nil {
        log.Fatalf("failed to close init DB connection: %v", err)
    }

    // Benchmark read-only workload for a fixed duration
    var wg sync.WaitGroup
    var totalOps uint64
    duration := 600 * time.Second
    endTime := time.Now().Add(duration)

    log.Printf("Starting read-only benchmark for %v...", duration)
    for _, addr := range servers {
        wg.Add(1)
        go func(endpoint string) {
            defer wg.Done()
            localDSN := fmt.Sprintf(dsnTemplate, endpoint)
            conn, err := sql.Open("mysql", localDSN)
            if err != nil {
                log.Printf("failed to open connection to %s: %v", endpoint, err)
                return
            }
            defer conn.Close()

            // configure session for stale reads if requested
            if *staleness != 0 {
                query := fmt.Sprintf("SET SESSION tidb_read_staleness = %d", *staleness)
                if _, err := conn.Exec(query); err != nil {
                    log.Printf("failed to set read staleness on %s: %v (query: %s)", endpoint, err, query)
                }
            }

            for time.Now().Before(endTime) {
                id := rand.Intn(10000) + 1
                rows, err := conn.Query("SELECT id, name, age FROM users WHERE id = ?", id)
                if err != nil {
                    log.Printf("query error on %s: %v", endpoint, err)
                    continue
                }
                rows.Close()
                atomic.AddUint64(&totalOps, 1)
            }
        }(addr)
    }
    wg.Wait()

    ops := atomic.LoadUint64(&totalOps)
    tps := float64(ops) / duration.Seconds()
    log.Printf("Total operations: %d, TPS=%.2f", ops, tps)
}
