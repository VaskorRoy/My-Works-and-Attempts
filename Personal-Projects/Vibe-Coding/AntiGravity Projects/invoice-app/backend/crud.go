package backend

import (
	"database/sql"
)

// Invoice represents a single invoice record with optional custom fields.
type Invoice struct {
	ID               int64             `json:"id"`
	VendorName       string            `json:"vendor_name"`
	InvoiceValue     float64           `json:"invoice_value"`
	InvoiceDate      string            `json:"invoice_date"`
	BillReceivedDate string            `json:"bill_received_date"`
	TaskOfInvoice    string            `json:"task_of_invoice"`
	VendorType       string            `json:"vendor_type"`
	CostType         string            `json:"cost_type"`
	RecommendDate    string            `json:"recommend_date"`
	DisbursementDate string            `json:"disbursement_date"`
	IsPaid           bool              `json:"is_paid"`
	NetValue         float64           `json:"net_value"`
	AIT              float64           `json:"ait"`
	VAT              float64           `json:"vat"`
	AITPercentage    float64           `json:"ait_percentage"`
	VATPercentage    float64           `json:"vat_percentage"`
	Currency         string            `json:"currency"`
	AttachmentPath   string            `json:"attachment_path"`
	CustomData       map[string]string `json:"custom_data"`
}

// ChartConfig represents a saved chart configuration.
type ChartConfig struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	GroupBy string `json:"group_by"`
	Metric  string `json:"metric"`
	IsPreset bool   `json:"is_preset"`
}

// AppSetting represents a key-value pair for application settings.
type AppSetting struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AddInvoice inserts a new invoice and its custom field values in a single transaction.
func (db *DB) AddInvoice(inv Invoice) (int64, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`INSERT INTO invoices (
		vendor_name, invoice_value, invoice_date,
		bill_received_date, task_of_invoice, vendor_type,
		cost_type, recommend_date, disbursement_date, is_paid,
		net_value, ait, vat, ait_percentage, vat_percentage, currency, attachment_path
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.VendorName, inv.InvoiceValue, inv.InvoiceDate,
		inv.BillReceivedDate, inv.TaskOfInvoice, inv.VendorType,
		inv.CostType, inv.RecommendDate, inv.DisbursementDate, inv.IsPaid,
		inv.NetValue, inv.AIT, inv.VAT, inv.AITPercentage, inv.VATPercentage,
		inv.Currency, inv.AttachmentPath,
	)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Insert custom field values — use tx for the lookup too (avoids connection contention)
	for fieldName, value := range inv.CustomData {
		if value == "" {
			continue
		}
		var fieldID int
		err := tx.QueryRow("SELECT id FROM custom_fields WHERE field_name = ?", fieldName).Scan(&fieldID)
		if err == nil {
			_, _ = tx.Exec("INSERT INTO invoice_custom_data (invoice_id, field_id, field_value) VALUES (?, ?, ?)", id, fieldID, value)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

// GetAllInvoices retrieves all invoices with their custom data.
// Always returns a non-nil slice (may be empty).
func (db *DB) GetAllInvoices() ([]Invoice, error) {
	invoices := make([]Invoice, 0)

	rows, err := db.conn.Query(`SELECT id, vendor_name, invoice_value, invoice_date,
		bill_received_date, task_of_invoice, vendor_type, cost_type,
		recommend_date, disbursement_date, is_paid, net_value, ait, vat, ait_percentage, vat_percentage, COALESCE(currency, 'USD'), COALESCE(attachment_path, '') FROM invoices ORDER BY id DESC`)
	if err != nil {
		return invoices, err
	}

	for rows.Next() {
		var inv Invoice
		if err := rows.Scan(&inv.ID, &inv.VendorName, &inv.InvoiceValue, &inv.InvoiceDate,
			&inv.BillReceivedDate, &inv.TaskOfInvoice, &inv.VendorType, &inv.CostType,
			&inv.RecommendDate, &inv.DisbursementDate, &inv.IsPaid, 
			&inv.NetValue, &inv.AIT, &inv.VAT, &inv.AITPercentage, &inv.VATPercentage, &inv.Currency, &inv.AttachmentPath); err != nil {
			rows.Close()
			return invoices, err
		}
		invoices = append(invoices, inv)
	}
	rows.Close()

	// Load custom data for each invoice after releasing the query cursor
	for i := range invoices {
		invoices[i].CustomData = make(map[string]string)
		customRows, err := db.conn.Query(`
			SELECT cf.field_name, icd.field_value
			FROM invoice_custom_data icd
			JOIN custom_fields cf ON icd.field_id = cf.id
			WHERE icd.invoice_id = ?`, invoices[i].ID)
		if err != nil {
			continue
		}
		for customRows.Next() {
			var name, val string
			if err := customRows.Scan(&name, &val); err == nil {
				invoices[i].CustomData[name] = val
			}
		}
		customRows.Close()
	}

	return invoices, nil
}

// MarkAsPaid toggles the paid status of an invoice.
func (db *DB) MarkAsPaid(id int64, paid bool) error {
	_, err := db.conn.Exec("UPDATE invoices SET is_paid = ? WHERE id = ?", paid, id)
	return err
}

// DeleteInvoice removes an invoice. Custom data is cascade-deleted via foreign keys.
func (db *DB) DeleteInvoice(id int64) error {
	_, err := db.conn.Exec("DELETE FROM invoices WHERE id = ?", id)
	return err
}

// AddCustomField registers a new dynamic field type.
func (db *DB) AddCustomField(name string, fieldType string) error {
	_, err := db.conn.Exec("INSERT INTO custom_fields (field_name, field_type) VALUES (?, ?)", name, fieldType)
	return err
}

// GetCustomFields returns all registered custom fields as a name→type map.
// Always returns a non-nil map (may be empty).
func (db *DB) GetCustomFields() (map[string]string, error) {
	fields := make(map[string]string)
	rows, err := db.conn.Query("SELECT field_name, field_type FROM custom_fields")
	if err != nil {
		return fields, err
	}
	defer rows.Close()

	for rows.Next() {
		var name, ftype string
		if err := rows.Scan(&name, &ftype); err == nil {
			fields[name] = ftype
		}
	}
	return fields, nil
}

// DeleteCustomField removes a custom field. Related invoice_custom_data is cascade-deleted.
func (db *DB) DeleteCustomField(name string) error {
	_, err := db.conn.Exec("DELETE FROM custom_fields WHERE field_name = ?", name)
	return err
}

// UpdateInvoice updates an existing invoice and completely replaces its custom data.
func (db *DB) UpdateInvoice(inv Invoice) error {
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`UPDATE invoices SET 
		vendor_name=?, invoice_value=?, invoice_date=?,
		bill_received_date=?, task_of_invoice=?, vendor_type=?,
		cost_type=?, recommend_date=?, disbursement_date=?, is_paid=?,
		net_value=?, ait=?, vat=?, ait_percentage=?, vat_percentage=?, currency=?, attachment_path=?
		WHERE id=?`,
		inv.VendorName, inv.InvoiceValue, inv.InvoiceDate,
		inv.BillReceivedDate, inv.TaskOfInvoice, inv.VendorType,
		inv.CostType, inv.RecommendDate, inv.DisbursementDate, inv.IsPaid,
		inv.NetValue, inv.AIT, inv.VAT, inv.AITPercentage, inv.VATPercentage,
		inv.Currency, inv.AttachmentPath, inv.ID)
	if err != nil {
		return err
	}

	// Replace old custom data
	_, err = tx.Exec("DELETE FROM invoice_custom_data WHERE invoice_id=?", inv.ID)
	if err != nil { return err }

	for fieldName, value := range inv.CustomData {
		if value == "" { continue }
		var fieldID int
		err := tx.QueryRow("SELECT id FROM custom_fields WHERE field_name = ?", fieldName).Scan(&fieldID)
		if err == nil {
			_, _ = tx.Exec("INSERT INTO invoice_custom_data (invoice_id, field_id, field_value) VALUES (?, ?, ?)", inv.ID, fieldID, value)
		}
	}

	return tx.Commit()
}

// ClearAllInvoices wipes all invoices. Useful for DB reset or import over-write.
func (db *DB) ClearAllInvoices() error {
	_, err := db.conn.Exec("DELETE FROM invoices")
	return err
}

// GetCategories returns dynamic categories. categoryType is either "vendor" or "cost".
func (db *DB) GetCategories(categoryType string) ([]string, error) {
	var cats []string
	query := "SELECT name FROM vendor_categories ORDER BY name"
	if categoryType == "cost" {
		query = "SELECT name FROM cost_categories ORDER BY name"
	}
	
	rows, err := db.conn.Query(query)
	if err != nil {
		return cats, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			cats = append(cats, name)
		}
	}
	return cats, nil
}

// AddCategory inserts a new dynamic category.
func (db *DB) AddCategory(categoryType string, name string) error {
	query := "INSERT INTO vendor_categories (name) VALUES (?)"
	if categoryType == "cost" {
		query = "INSERT INTO cost_categories (name) VALUES (?)"
	}
	_, err := db.conn.Exec(query, name)
	return err
}

// RenameCategory modifies an existing category name.
func (db *DB) RenameCategory(categoryType string, oldName string, newName string) error {
	query := "UPDATE vendor_categories SET name = ? WHERE name = ?"
	if categoryType == "cost" {
		query = "UPDATE cost_categories SET name = ? WHERE name = ?"
	}
	_, err := db.conn.Exec(query, newName, oldName)
	return err
}

// DeleteCategory removes an existing category entirely.
func (db *DB) DeleteCategory(categoryType string, name string) error {
	query := "DELETE FROM vendor_categories WHERE name = ?"
	if categoryType == "cost" {
		query = "DELETE FROM cost_categories WHERE name = ?"
	}
	_, err := db.conn.Exec(query, name)
	return err
}

// Vendor represents a whitelisted vendor
type Vendor struct {
	ID              int64  `json:"id"`
	Name            string `json:"name"`
	TaxID           string `json:"tax_id"`
	DefaultCategory string `json:"default_category"`
}

// GetVendors returns all registered vendors
func (db *DB) GetVendors() ([]Vendor, error) {
	vendors := []Vendor{}
	rows, err := db.conn.Query("SELECT id, name, COALESCE(tax_id, ''), COALESCE(default_category, '') FROM vendors ORDER BY name")
	if err != nil {
		return vendors, err
	}
	defer rows.Close()
	for rows.Next() {
		var v Vendor
		if err := rows.Scan(&v.ID, &v.Name, &v.TaxID, &v.DefaultCategory); err == nil {
			vendors = append(vendors, v)
		}
	}
	return vendors, nil
}

// AddVendor inserts a new vendor to the whitelist
func (db *DB) AddVendor(v Vendor) error {
	_, err := db.conn.Exec("INSERT INTO vendors (name, tax_id, default_category) VALUES (?, ?, ?)", v.Name, v.TaxID, v.DefaultCategory)
	return err
}

// DeleteVendor removes a vendor from the whitelist
func (db *DB) DeleteVendor(id int64) error {
	_, err := db.conn.Exec("DELETE FROM vendors WHERE id = ?", id)
	return err
}

// GetChartConfigs returns all saved chart configurations.
func (db *DB) GetChartConfigs() ([]ChartConfig, error) {
	configs := []ChartConfig{}
	rows, err := db.conn.Query("SELECT id, name, type, group_by, metric, is_preset FROM chart_configs ORDER BY id DESC")
	if err != nil {
		return configs, err
	}
	defer rows.Close()
	for rows.Next() {
		var c ChartConfig
		if err := rows.Scan(&c.ID, &c.Name, &c.Type, &c.GroupBy, &c.Metric, &c.IsPreset); err == nil {
			configs = append(configs, c)
		}
	}
	return configs, nil
}

// SaveChartConfig persists a new or updated chart configuration.
func (db *DB) SaveChartConfig(c ChartConfig) error {
	if c.ID > 0 {
		_, err := db.conn.Exec("UPDATE chart_configs SET name=?, type=?, group_by=?, metric=?, is_preset=? WHERE id=?",
			c.Name, c.Type, c.GroupBy, c.Metric, c.IsPreset, c.ID)
		return err
	}
	_, err := db.conn.Exec("INSERT INTO chart_configs (name, type, group_by, metric, is_preset) VALUES (?, ?, ?, ?, ?)",
		c.Name, c.Type, c.GroupBy, c.Metric, c.IsPreset)
	return err
}

// DeleteChartConfig removes a saved chart configuration.
func (db *DB) DeleteChartConfig(id int64) error {
	_, err := db.conn.Exec("DELETE FROM chart_configs WHERE id = ?", id)
	return err
}

// GetSetting retrieves a single setting value.
func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.conn.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}

// UpdateSetting saves or updates a setting.
func (db *DB) UpdateSetting(key string, value string) error {
	_, err := db.conn.Exec("INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value", key, value)
	return err
}
