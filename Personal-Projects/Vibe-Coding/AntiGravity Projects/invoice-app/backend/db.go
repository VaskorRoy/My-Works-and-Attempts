package backend

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	conn *sql.DB
}

// InitDB ensures the local database file exists in user config space.
func InitDB() *DB {
	configDir, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Could not determine config directory:", err)
	}
	appDir := filepath.Join(configDir, "invoice-app")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		log.Fatal("Could not create app directory:", err)
	}

	dbPath := filepath.Join(appDir, "invoices.db")

	// Enable WAL mode and foreign keys via connection string
	conn, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_foreign_keys=on")
	if err != nil {
		log.Fatal(err)
	}

	// Limit to 1 connection to avoid SQLite locking issues
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn}
	db.createTables()
	return db
}

func (db *DB) createTables() {
	query := `
	CREATE TABLE IF NOT EXISTS invoices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vendor_name TEXT NOT NULL DEFAULT '',
		invoice_value REAL NOT NULL DEFAULT 0,
		invoice_date TEXT NOT NULL DEFAULT '',
		bill_received_date TEXT NOT NULL DEFAULT '',
		task_of_invoice TEXT NOT NULL DEFAULT '',
		vendor_type TEXT NOT NULL DEFAULT '',
		cost_type TEXT NOT NULL DEFAULT '',
		recommend_date TEXT NOT NULL DEFAULT '',
		disbursement_date TEXT NOT NULL DEFAULT '',
		is_paid BOOLEAN NOT NULL DEFAULT 0,
		net_value REAL NOT NULL DEFAULT 0,
		ait REAL NOT NULL DEFAULT 0,
		vat REAL NOT NULL DEFAULT 0,
		ait_percentage REAL NOT NULL DEFAULT 0,
		vat_percentage REAL NOT NULL DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS custom_fields (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		field_name TEXT UNIQUE NOT NULL,
		field_type TEXT NOT NULL DEFAULT 'TEXT'
	);

	CREATE TABLE IF NOT EXISTS invoice_custom_data (
		invoice_id INTEGER NOT NULL,
		field_id INTEGER NOT NULL,
		field_value TEXT NOT NULL DEFAULT '',
		FOREIGN KEY(invoice_id) REFERENCES invoices(id) ON DELETE CASCADE,
		FOREIGN KEY(field_id) REFERENCES custom_fields(id) ON DELETE CASCADE,
		PRIMARY KEY(invoice_id, field_id)
	);

	CREATE TABLE IF NOT EXISTS vendor_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS cost_categories (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL
	);

	CREATE TABLE IF NOT EXISTS chart_configs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		group_by TEXT NOT NULL,
		metric TEXT NOT NULL,
		is_preset BOOLEAN DEFAULT 0
	);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`

	_, err := db.conn.Exec(query)
	if err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// Run defensive migrations for existing databases to add new columns
	// Schema Migrations block to ensure backward compatibility
	migrations := []string{
		"ALTER TABLE invoices ADD COLUMN net_value REAL NOT NULL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN ait REAL NOT NULL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN vat REAL NOT NULL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN ait_percentage REAL NOT NULL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN vat_percentage REAL NOT NULL DEFAULT 0",
		"ALTER TABLE invoices ADD COLUMN currency TEXT DEFAULT 'USD'",
	}
	
	for _, m := range migrations {
		_, _ = db.conn.Exec(m) // Ignore constraint errors from existing columns
	}

	// Seed vendor categories
	db.seedCategories("vendor_categories", []string{"Supplier", "Contractor", "Software/SaaS", "Utility", "Consulting", "Other"})
	// Seed cost categories
	db.seedCategories("cost_categories", []string{"Operational", "Capital", "R&D", "Marketing", "Administrative", "Other"})
	// Vendor Whitelist / Book
	vendorTable := `
	CREATE TABLE IF NOT EXISTS vendors (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		tax_id TEXT,
		default_category TEXT
	);`
	_, _ = db.conn.Exec(vendorTable)

	// Add PDF attachment metadata column to invoices
	db.migrateAddColumn("invoices", "attachment_path", "TEXT DEFAULT ''")

	fmt.Println("Database initialized successfully.")
}

// seedCategories helper to prepopulate if empty
func (db *DB) seedCategories(table string, defaults []string) {
	var count int
	queryCheck := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
	err := db.conn.QueryRow(queryCheck).Scan(&count)
	if err == nil && count == 0 {
		for _, cat := range defaults {
			queryInsert := fmt.Sprintf("INSERT INTO %s (name) VALUES (?)", table)
			_, _ = db.conn.Exec(queryInsert, cat)
		}
	}
}

// migrateAddColumn safely adds a column if it doesn't already exist.
func (db *DB) migrateAddColumn(table string, column string, def string) {
	query := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, def)
	_, _ = db.conn.Exec(query) // Ignore error if column already exists
}
