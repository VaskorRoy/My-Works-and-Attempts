package backend

import (
	"fmt"

	"github.com/gen2brain/beeep"
)

// CheckUnpaidInvoices uses a targeted SQL query to count overdue invoices
// and sends an OS notification if any are found. Avoids loading all invoices.
func (db *DB) CheckUnpaidInvoices() error {
	var count int
	err := db.conn.QueryRow(`
		SELECT COUNT(*) FROM invoices
		WHERE is_paid = 0
		  AND bill_received_date != ''
		  AND julianday('now') - julianday(bill_received_date) > 14
	`).Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return db.Notify(
			"Invoice Alert",
			fmt.Sprintf("You have %d unpaid invoices overdue (>14 days).", count),
		)
	}
	return nil
}

// Notify sends a generic OS notification.
func (db *DB) Notify(title string, message string) error {
	return beeep.Notify(title, message, "")
}
