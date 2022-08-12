package db

import (
	"context"
	"database/sql"
	"fmt"
)

// Store provides all functions to execute the db queries and transactions
type Store struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

// execTx executes a function within a database transaction
func (store *Store) execTx(ctx context.Context, fn func(queries *Queries) error) error {
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	q := New(tx)
	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			fmt.Errorf("tx err: %v, rb error: %v", err, rbErr)
		}
		return err
	}
	return tx.Commit()
}

// TransferTxParams contains the input parameter of the transfer transaction
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID   int64 `json:"to_account_id"`
	Amount        int64 `json:"amount"`
}

// TransferTxResult is the result of the transfer transaction
type TransferTxResult struct {
	Transfer    Transfer `json:"transfer"`
	FromAccount Account  `json:"from_account"`
	ToAccount   Account  `json:"to_account"`
	FromEntry   Entry    `json:"from_entry"`
	ToEntry     Entry    `json:"to_entry"`
}

var txKey = "myKey"

// TransferTx performs a money transfer from one account to the other.
// It creates a transfer record, add account entries, and updates accounts balance within a single db transaction
func (store *Store) TransferTx(ctx context.Context, arg TransferTxParams) (TransferTxResult, error) {
	var result TransferTxResult

	err := store.execTx(ctx, func(queries *Queries) error {
		var err error

		txName, _ := ctx.Value(txKey).(*string)

		println(*txName, "Create transfer")
		result.Transfer, err = queries.CreateTransfer(ctx, CreateTransferParams{
			FromAccountID: arg.FromAccountID,
			ToAccountID:   arg.ToAccountID,
			Amount:        arg.Amount,
		})
		if err != nil {
			return err
		}

		println(*txName, "Create entry 1")
		result.FromEntry, err = queries.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.FromAccountID,
			Amount:    -arg.Amount,
		})
		if err != nil {
			return err
		}

		println(*txName, "Create entry 2")
		result.ToEntry, err = queries.CreateEntry(ctx, CreateEntryParams{
			AccountID: arg.ToAccountID,
			Amount:    +arg.Amount,
		})
		if err != nil {
			return err
		}

		// TODO updates accounts balance
		result.FromAccount, err = queries.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.FromAccountID,
			Amount: -arg.Amount,
		})

		println(*txName, "Update ToAccount")
		result.ToAccount, err = queries.AddAccountBalance(ctx, AddAccountBalanceParams{
			ID:     arg.ToAccountID,
			Amount: arg.Amount,
		})

		return nil
	})

	return result, err
}
