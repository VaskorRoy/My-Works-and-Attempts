package main

import (
	"context"
	"invoice-app/backend"
)

// App struct
type App struct {
	ctx context.Context
	DB  *backend.DB
}

// NewApp creates a new App application struct
func NewApp() *App {
	// Initialize the embedded SQLite DB
	db := backend.InitDB()
	return &App{
		DB: db,
	}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	// Check and fire OS OS push notification for 14-day unpaid overdue invoices
	go a.DB.CheckUnpaidInvoices()
}
