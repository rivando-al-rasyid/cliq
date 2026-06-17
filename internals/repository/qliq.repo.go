package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/model"
	"github.com/rivando-al-rasyid/vanwallet-backend/internals/pkg"
)

type TransactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepo(db *pgxpool.Pool) *TransactionRepo {
	return &TransactionRepo{db: db}
}

// VerifyPIN fetches the user's stored argon2 PIN hash and compares it with rawPin.
// Returns nil on success, error on mismatch or PIN not set.
func (t *TransactionRepo) VerifyPIN(ctx context.Context, email, rawPin string) error {
	var pinHash string
	err := t.db.QueryRow(ctx, `
		SELECT COALESCE(up.pin_hash, '')
		FROM user_pins up
		JOIN users u ON up.user_id = u.id
		WHERE u.email = $1`, email,
	).Scan(&pinHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("pin not set")
		}
		return fmt.Errorf("VerifyPIN query: %w", err)
	}
	if pinHash == "" {
		return errors.New("pin not set")
	}
	var hc pkg.HashConfig
	hc.UseRecommended()
	if err := hc.Compare(rawPin, pinHash); err != nil {
		return errors.New("invalid pin")
	}
	return nil
}

// WalletBelongsToUser returns true when walletID is owned by the user identified by email.
func (t *TransactionRepo) WalletBelongsToUser(ctx context.Context, email string, walletID uuid.UUID) (bool, error) {
	var exists bool
	err := t.db.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM wallets w
			JOIN users u ON w.user_id = u.id
			WHERE u.email = $1 AND w.id = $2
		)`, email, walletID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("WalletBelongsToUser: %w", err)
	}
	return exists, nil
}

// GetSummary returns aggregated balance, income, expense, and per-wallet breakdown.
func (t *TransactionRepo) GetSummary(ctx context.Context, email string) (model.TransactionSummary, error) {
	aggSQL := `
	WITH TargetUser AS (
		SELECT id FROM users WHERE email = $1
	),
	UserWallets AS (
		SELECT id, balance FROM wallets WHERE user_id IN (SELECT id FROM TargetUser)
	),
	WalletAgg AS (
		SELECT COALESCE(SUM(balance), 0) AS current_balance FROM UserWallets
	),
	TransactionAgg AS (
		SELECT
			COALESCE(SUM(CASE WHEN type = 'TRANSFER_IN'  AND status = 'SUCCESS' THEN amount ELSE 0 END), 0) AS total_income,
			COALESCE(SUM(CASE WHEN type IN ('EXPENSE','WITHDRAWAL','TRANSFER_OUT') AND status = 'SUCCESS' THEN (amount + admin_fee) ELSE 0 END), 0) AS total_expense
		FROM transactions
		WHERE wallet_id IN (SELECT id FROM UserWallets)
	)
	SELECT wa.current_balance, ta.total_income, ta.total_expense
	FROM WalletAgg wa CROSS JOIN TransactionAgg ta`

	var s model.TransactionSummary
	if err := t.db.QueryRow(ctx, aggSQL, email).Scan(
		&s.CurrentBalance, &s.TotalIncome, &s.TotalExpense,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.TransactionSummary{}, errors.New("user not found")
		}
		return model.TransactionSummary{}, fmt.Errorf("GetSummary agg: %w", err)
	}

	rows, err := t.db.Query(ctx, `
		SELECT w.id, w.label, w.balance
		FROM wallets w
		JOIN users u ON w.user_id = u.id
		WHERE u.email = $1
		ORDER BY w.created_at ASC`, email)
	if err != nil {
		return model.TransactionSummary{}, fmt.Errorf("GetSummary wallets: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var w model.WalletSummary
		if err := rows.Scan(&w.ID, &w.Label, &w.Balance); err != nil {
			return model.TransactionSummary{}, fmt.Errorf("GetSummary wallet scan: %w", err)
		}
		s.Wallets = append(s.Wallets, w)
	}
	return s, rows.Err()
}

// GetTransactionReport returns daily (7days) or weekly (30days) chart buckets.
func (t *TransactionRepo) GetTransactionReport(ctx context.Context, email, rangeParam, typeFilter string) ([]model.ChartPoint, error) {
	var dateTrunc, dateLabel, interval string
	switch rangeParam {
	case "30days":
		dateTrunc = "week"
		dateLabel = "'W' || TO_CHAR(DATE_TRUNC('week', created_at), 'IW')"
		interval = "30 days"
	default:
		dateTrunc = "day"
		dateLabel = "TO_CHAR(DATE_TRUNC('day', created_at), 'Dy')"
		interval = "7 days"
	}
	var incomeExpr, expenseExpr string
	switch typeFilter {
	case "income":
		incomeExpr = "COALESCE(SUM(CASE WHEN type = 'TRANSFER_IN' AND status = 'SUCCESS' THEN amount ELSE 0 END), 0)"
		expenseExpr = "0"
	case "expense":
		incomeExpr = "0"
		expenseExpr = "COALESCE(SUM(CASE WHEN type IN ('EXPENSE','WITHDRAWAL','TRANSFER_OUT') AND status = 'SUCCESS' THEN (amount + admin_fee) ELSE 0 END), 0)"
	default:
		incomeExpr = "COALESCE(SUM(CASE WHEN type = 'TRANSFER_IN' AND status = 'SUCCESS' THEN amount ELSE 0 END), 0)"
		expenseExpr = "COALESCE(SUM(CASE WHEN type IN ('EXPENSE','WITHDRAWAL','TRANSFER_OUT') AND status = 'SUCCESS' THEN (amount + admin_fee) ELSE 0 END), 0)"
	}
	sql := fmt.Sprintf(`
		WITH UserWallets AS (
			SELECT w.id FROM wallets w JOIN users u ON w.user_id = u.id WHERE u.email = $1
		)
		SELECT %s AS label, %s AS income, %s AS expense
		FROM transactions
		WHERE wallet_id IN (SELECT id FROM UserWallets)
		  AND created_at >= NOW() - INTERVAL '%s'
		GROUP BY DATE_TRUNC('%s', created_at)
		ORDER BY DATE_TRUNC('%s', created_at)`,
		dateLabel, incomeExpr, expenseExpr, interval, dateTrunc, dateTrunc,
	)
	rows, err := t.db.Query(ctx, sql, email)
	if err != nil {
		return nil, fmt.Errorf("GetTransactionReport: %w", err)
	}
	defer rows.Close()
	var result []model.ChartPoint
	for rows.Next() {
		var cp model.ChartPoint
		if err := rows.Scan(&cp.Label, &cp.Income, &cp.Expense); err != nil {
			return nil, fmt.Errorf("GetTransactionReport scan: %w", err)
		}
		result = append(result, cp)
	}
	return result, rows.Err()
}

func (t *TransactionRepo) GetAllHistory(ctx context.Context, email string, filter model.HistoryFilter) ([]model.HistoryItem, int, error) {
	offset := (filter.Page - 1) * filter.Limit

	baseSQL := `
		WITH UserWallets AS (
			SELECT w.id, w.label
			FROM wallets w
			JOIN users u ON w.user_id = u.id
			WHERE u.email = $1
		), UnifiedHistory AS (
			SELECT
				tr.id::text                         AS id,
				'transaction'                       AS source,
				tr.type::text                       AS type,
				CASE
					WHEN tr.type = 'TRANSFER_IN' THEN 'income'
					ELSE 'expense'
				END                                AS direction,
				tr.amount                           AS amount,
				tr.admin_fee                        AS admin_fee,
				tr.status::text                     AS status,
				''                                  AS payment_method,
				COALESCE(tr.note, '')               AS note,
				tr.wallet_id::text                  AS wallet_id,
				uw.label                            AS wallet_label,
				tr.created_at                       AS created_at
			FROM transactions tr
			JOIN UserWallets uw ON tr.wallet_id = uw.id

			UNION ALL

			SELECT
				tp.id::text                         AS id,
				'topup'                             AS source,
				'TOPUP'                             AS type,
				'income'                            AS direction,
				tp.amount                           AS amount,
				0                                   AS admin_fee,
				tp.status::text                     AS status,
				COALESCE(tp.payment_method::text, '') AS payment_method,
				''                                  AS note,
				tp.wallet_id::text                  AS wallet_id,
				uw.label                            AS wallet_label,
				tp.created_at                       AS created_at
			FROM topups tp
			JOIN UserWallets uw ON tp.wallet_id = uw.id
		)
		SELECT * FROM UnifiedHistory`

	args := []any{email}
	conditions := make([]string, 0)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if filter.WalletID != "" {
		conditions = append(conditions, "wallet_id = "+addArg(filter.WalletID))
	}
	if filter.Source != "" {
		conditions = append(conditions, "LOWER(source) = LOWER("+addArg(filter.Source)+")")
	}
	if filter.Type != "" {
		conditions = append(conditions, "UPPER(type) = UPPER("+addArg(filter.Type)+")")
	}
	if filter.Status != "" {
		conditions = append(conditions, "UPPER(status) = UPPER("+addArg(filter.Status)+")")
	}
	if filter.Direction != "" {
		conditions = append(conditions, "LOWER(direction) = LOWER("+addArg(filter.Direction)+")")
	}
	if filter.StartDate != "" {
		conditions = append(conditions, "created_at >= "+addArg(filter.StartDate)+"::date")
	}
	if filter.EndDate != "" {
		conditions = append(conditions, "created_at < ("+addArg(filter.EndDate)+"::date + INTERVAL '1 day')")
	}
	if filter.Query != "" {
		q := "%" + filter.Query + "%"
		placeholder := addArg(q)
		conditions = append(conditions, `(
			id ILIKE `+placeholder+` OR
			source ILIKE `+placeholder+` OR
			type ILIKE `+placeholder+` OR
			direction ILIKE `+placeholder+` OR
			status ILIKE `+placeholder+` OR
			payment_method ILIKE `+placeholder+` OR
			note ILIKE `+placeholder+` OR
			wallet_id ILIKE `+placeholder+` OR
			wallet_label ILIKE `+placeholder+`
		)`)
	}

	whereSQL := ""
	if len(conditions) > 0 {
		whereSQL = " WHERE " + strings.Join(conditions, " AND ")
	}

	countSQL := `SELECT COUNT(*) FROM (` + baseSQL + whereSQL + `) AS filtered_history`
	var total int
	if err := t.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("GetAllHistory count: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, filter.Limit, offset)
	querySQL := baseSQL + whereSQL + fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", len(queryArgs)-1, len(queryArgs))

	rows, err := t.db.Query(ctx, querySQL, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("GetAllHistory query: %w", err)
	}
	defer rows.Close()

	var items []model.HistoryItem
	for rows.Next() {
		var h model.HistoryItem
		if err := rows.Scan(
			&h.ID, &h.Source, &h.Type, &h.Direction, &h.Amount, &h.AdminFee,
			&h.Status, &h.PaymentMethod, &h.Note, &h.WalletID, &h.WalletLabel, &h.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("GetAllHistory scan: %w", err)
		}
		items = append(items, h)
	}
	return items, total, rows.Err()
}

// CreateTopup creates a PENDING topup record.
func (t *TransactionRepo) CreateTopup(ctx context.Context, req model.Topup) (model.Topup, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return model.Topup{}, fmt.Errorf("CreateTopup begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var topup model.Topup
	err = tx.QueryRow(ctx, `
		INSERT INTO topups (wallet_id, amount, status, payment_method)
		VALUES ($1, $2, $3, $4)
		RETURNING id, wallet_id, amount, status, payment_method, external_reference, created_at`,
		req.WalletID, req.Amount, model.TransactionStatusPending, req.PaymentMethod,
	).Scan(&topup.ID, &topup.WalletID, &topup.Amount, &topup.Status, &topup.PaymentMethod, &topup.ExternalReference, &topup.CreatedAt)
	if err != nil {
		return model.Topup{}, fmt.Errorf("CreateTopup insert: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Topup{}, fmt.Errorf("CreateTopup commit: %w", err)
	}
	return topup, nil
}

// ConfirmTopup sets topup to SUCCESS and credits the wallet atomically.
func (t *TransactionRepo) ConfirmTopup(ctx context.Context, email string, topupID uuid.UUID) (model.Topup, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return model.Topup{}, fmt.Errorf("ConfirmTopup begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var topup model.Topup
	err = tx.QueryRow(ctx, `
		UPDATE topups tp
		SET status = 'SUCCESS', updated_at = now()
		FROM wallets w
		JOIN users u ON w.user_id = u.id
		WHERE tp.wallet_id = w.id
		  AND tp.id = $1
		  AND u.email = $2
		  AND tp.status = 'PENDING'
		RETURNING tp.id, tp.wallet_id, tp.amount, tp.status, tp.payment_method, tp.external_reference, tp.created_at`,
		topupID, email,
	).Scan(&topup.ID, &topup.WalletID, &topup.Amount, &topup.Status, &topup.PaymentMethod, &topup.ExternalReference, &topup.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.Topup{}, errors.New("topup not found or already processed")
		}
		return model.Topup{}, fmt.Errorf("ConfirmTopup update: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE wallets SET balance = balance + $1, updated_at = now() WHERE id = $2`,
		topup.Amount, topup.WalletID,
	); err != nil {
		return model.Topup{}, fmt.Errorf("ConfirmTopup credit wallet: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Topup{}, fmt.Errorf("ConfirmTopup commit: %w", err)
	}
	return topup, nil
}

// CreateWithdrawal inserts a WITHDRAWAL transaction + detail, debits wallet atomically.
func (t *TransactionRepo) CreateWithdrawal(ctx context.Context, walletID uuid.UUID, amount, adminFee int64, bank model.Withdrawal) (model.Transaction, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var balance int64
	if err = tx.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`, walletID).Scan(&balance); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal lock wallet: %w", err)
	}
	if balance < amount+adminFee {
		return model.Transaction{}, errors.New("insufficient balance")
	}
	var txRow model.Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (wallet_id, type, amount, admin_fee, status)
		VALUES ($1, 'WITHDRAWAL', $2, $3, 'PENDING')
		RETURNING id, wallet_id, type, amount, admin_fee, status, idempotency_key, note, created_at, updated_at`,
		walletID, amount, adminFee,
	).Scan(&txRow.ID, &txRow.WalletID, &txRow.Type, &txRow.Amount, &txRow.AdminFee, &txRow.Status, &txRow.IdempotencyKey, &txRow.Note, &txRow.CreatedAt, &txRow.UpdatedAt)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal insert transaction: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`INSERT INTO withdrawals (transaction_id, bank_name, account_number, account_holder) VALUES ($1, $2, $3, $4)`,
		txRow.ID, bank.BankName, bank.AccountNumber, bank.AccountHolder,
	); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal insert detail: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE wallets SET balance = balance - $1, updated_at = now() WHERE id = $2`,
		amount+adminFee, walletID,
	); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal debit wallet: %w", err)
	}
	if _, err = tx.Exec(ctx,
		`UPDATE transactions SET status = 'SUCCESS', updated_at = now() WHERE id = $1`, txRow.ID,
	); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal update status: %w", err)
	}
	txRow.Status = model.TransactionStatusSuccess
	if err = tx.Commit(ctx); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateWithdrawal commit: %w", err)
	}
	return txRow, nil
}

// CreateTransfer executes a peer-to-peer transfer atomically.
// Returns (transfer, senderTx, recipientTx, error). Controller sends only senderTx to caller.
func (t *TransactionRepo) CreateTransfer(ctx context.Context, senderWalletID, recipientWalletID uuid.UUID, amount, adminFee int64, note *string) (model.Transfer, model.Transaction, model.Transaction, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var senderBalance int64
	if err = tx.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`, senderWalletID).Scan(&senderBalance); err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer lock sender: %w", err)
	}
	if senderBalance < amount+adminFee {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, errors.New("insufficient balance")
	}
	var lockedRecipientID uuid.UUID
	if err = tx.QueryRow(ctx, `SELECT id FROM wallets WHERE id = $1 FOR UPDATE`, recipientWalletID).Scan(&lockedRecipientID); err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer lock recipient: %w", err)
	}

	var senderTx model.Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (wallet_id, type, amount, admin_fee, status, note)
		VALUES ($1, 'TRANSFER_OUT', $2, $3, 'SUCCESS', $4)
		RETURNING id, wallet_id, type, amount, admin_fee, status, idempotency_key, note, created_at, updated_at`,
		senderWalletID, amount, adminFee, note,
	).Scan(&senderTx.ID, &senderTx.WalletID, &senderTx.Type, &senderTx.Amount, &senderTx.AdminFee, &senderTx.Status, &senderTx.IdempotencyKey, &senderTx.Note, &senderTx.CreatedAt, &senderTx.UpdatedAt)
	if err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer TRANSFER_OUT: %w", err)
	}

	var recipientTx model.Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (wallet_id, type, amount, admin_fee, status, note)
		VALUES ($1, 'TRANSFER_IN', $2, 0, 'SUCCESS', $3)
		RETURNING id, wallet_id, type, amount, admin_fee, status, idempotency_key, note, created_at, updated_at`,
		recipientWalletID, amount, note,
	).Scan(&recipientTx.ID, &recipientTx.WalletID, &recipientTx.Type, &recipientTx.Amount, &recipientTx.AdminFee, &recipientTx.Status, &recipientTx.IdempotencyKey, &recipientTx.Note, &recipientTx.CreatedAt, &recipientTx.UpdatedAt)
	if err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer TRANSFER_IN: %w", err)
	}

	transferCode := fmt.Sprintf("TRF-%s", senderTx.ID.String()[:8])
	var transfer model.Transfer
	err = tx.QueryRow(ctx, `
		INSERT INTO transfers (transaction_id, recipient_transaction_id, transfer_code)
		VALUES ($1, $2, $3)
		RETURNING transaction_id, recipient_transaction_id, transfer_code, created_at`,
		senderTx.ID, recipientTx.ID, transferCode,
	).Scan(&transfer.TransactionID, &transfer.RecipientTransactionID, &transfer.TransferCode, &transfer.CreatedAt)
	if err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer link: %w", err)
	}

	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance - $1, updated_at = now() WHERE id = $2`, amount+adminFee, senderWalletID); err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer debit: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance + $1, updated_at = now() WHERE id = $2`, amount, recipientWalletID); err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer credit: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Transfer{}, model.Transaction{}, model.Transaction{}, fmt.Errorf("CreateTransfer commit: %w", err)
	}
	return transfer, senderTx, recipientTx, nil
}

// CreateExpense inserts an EXPENSE transaction + detail, debits wallet atomically.
func (t *TransactionRepo) CreateExpense(ctx context.Context, walletID uuid.UUID, amount, adminFee int64, category, merchantName, note *string) (model.Transaction, error) {
	tx, err := t.db.Begin(ctx)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	var balance int64
	if err = tx.QueryRow(ctx, `SELECT balance FROM wallets WHERE id = $1 FOR UPDATE`, walletID).Scan(&balance); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense lock wallet: %w", err)
	}
	if balance < amount+adminFee {
		return model.Transaction{}, errors.New("insufficient balance")
	}
	var txRow model.Transaction
	err = tx.QueryRow(ctx, `
		INSERT INTO transactions (wallet_id, type, amount, admin_fee, status, note)
		VALUES ($1, 'EXPENSE', $2, $3, 'SUCCESS', $4)
		RETURNING id, wallet_id, type, amount, admin_fee, status, idempotency_key, note, created_at, updated_at`,
		walletID, amount, adminFee, note,
	).Scan(&txRow.ID, &txRow.WalletID, &txRow.Type, &txRow.Amount, &txRow.AdminFee, &txRow.Status, &txRow.IdempotencyKey, &txRow.Note, &txRow.CreatedAt, &txRow.UpdatedAt)
	if err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense insert transaction: %w", err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO expenses (transaction_id, category, merchant_name) VALUES ($1, $2, $3)`, txRow.ID, category, merchantName); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense insert detail: %w", err)
	}
	if _, err = tx.Exec(ctx, `UPDATE wallets SET balance = balance - $1, updated_at = now() WHERE id = $2`, amount+adminFee, walletID); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense debit wallet: %w", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return model.Transaction{}, fmt.Errorf("CreateExpense commit: %w", err)
	}
	return txRow, nil
}

// SearchReceivers lists every transferable user wallet, with optional search.
func (t *TransactionRepo) SearchReceivers(
	ctx context.Context,
	callerEmail,
	query string,
	page,
	limit int,
) ([]model.ReceiverResult, int, error) {
	offset := (page - 1) * limit

	baseSQL := `
		FROM users u
		LEFT JOIN profiles p ON p.user_id = u.id
		JOIN wallets w ON w.user_id = u.id
		WHERE u.email != $1
	`

	args := []any{callerEmail}
	conditions := make([]string, 0)
	addArg := func(value any) string {
		args = append(args, value)
		return fmt.Sprintf("$%d", len(args))
	}

	if query != "" {
		placeholder := addArg("%" + query + "%")
		conditions = append(conditions, `(
			COALESCE(p.full_name, '') ILIKE `+placeholder+` OR
			COALESCE(p.phone, '') ILIKE `+placeholder+` OR
			u.email ILIKE `+placeholder+` OR
			w.label ILIKE `+placeholder+` OR
			w.id::text ILIKE `+placeholder+`
		)`)
	}

	whereSQL := ""
	if len(conditions) > 0 {
		whereSQL = " AND " + strings.Join(conditions, " AND ")
	}

	var total int
	countSQL := `SELECT COUNT(*) ` + baseSQL + whereSQL
	if err := t.db.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("SearchReceivers count: %w", err)
	}

	queryArgs := append([]any{}, args...)
	queryArgs = append(queryArgs, limit, offset)
	dataSQL := `
		SELECT u.id, u.email, p.full_name, p.phone, p.photo, w.id, w.label
	` + baseSQL + whereSQL + fmt.Sprintf(`
		ORDER BY COALESCE(p.full_name, u.email) ASC, w.label ASC
		LIMIT $%d OFFSET $%d`, len(queryArgs)-1, len(queryArgs))

	rows, err := t.db.Query(ctx, dataSQL, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("SearchReceivers query: %w", err)
	}
	defer rows.Close()

	var results []model.ReceiverResult
	for rows.Next() {
		var r model.ReceiverResult
		if err := rows.Scan(
			&r.UserID,
			&r.Email,
			&r.FullName,
			&r.Phone,
			&r.Photo,
			&r.WalletID,
			&r.WalletLabel,
		); err != nil {
			return nil, 0, fmt.Errorf("SearchReceivers scan: %w", err)
		}

		results = append(results, r)
	}

	return results, total, rows.Err()
}
