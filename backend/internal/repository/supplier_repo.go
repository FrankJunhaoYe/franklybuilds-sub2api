package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	dbaccount "github.com/Wei-Shaw/sub2api/ent/account"
	dbsupplier "github.com/Wei-Shaw/sub2api/ent/supplier"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type supplierRepository struct {
	client   *dbent.Client
	sql      sqlExecutor
	accounts *accountRepository
}

func NewSupplierRepository(client *dbent.Client, sqlDB *sql.DB, cache service.SchedulerCache) service.SupplierRepository {
	return &supplierRepository{client: client, sql: sqlDB, accounts: newAccountRepositoryWithSQL(client, sqlDB, cache)}
}

func supplierEntityToService(m *dbent.Supplier) *service.Supplier {
	return &service.Supplier{
		ID: m.ID, Name: m.Name, BaseURL: m.BaseURL, Status: m.Status, Notes: m.Notes,
		BalanceAccountID: m.BalanceAccountID, BalanceAPIKey: m.BalanceAPIKey,
		HasBalanceAPIKey: m.BalanceAPIKey != "", Balance: m.Balance,
		BalanceStatus: m.BalanceStatus, BalanceError: m.BalanceError,
		BalanceUpdatedAt: m.BalanceUpdatedAt, BalanceCheckedAt: m.BalanceCheckedAt,
		Revision: m.Revision, BalanceProbeSequence: m.BalanceProbeSequence,
		CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt,
	}
}

func (r *supplierRepository) Create(ctx context.Context, in *service.Supplier) error {
	m, err := r.client.Supplier.Create().SetName(in.Name).SetBaseURL(in.BaseURL).
		SetStatus(in.Status).SetNotes(in.Notes).SetBalanceAPIKey(in.BalanceAPIKey).
		SetNillableBalanceAccountID(in.BalanceAccountID).Save(ctx)
	if err != nil {
		return err
	}
	*in = *supplierEntityToService(m)
	return nil
}

func (r *supplierRepository) GetByID(ctx context.Context, id int64) (*service.Supplier, error) {
	m, err := r.client.Supplier.Get(ctx, id)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	out := supplierEntityToService(m)
	count, err := r.client.Account.Query().Where(dbaccount.SupplierIDEQ(id)).Count(ctx)
	if err != nil {
		return nil, err
	}
	out.AccountCount = int64(count)
	return out, nil
}

func (r *supplierRepository) List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]service.Supplier, *pagination.PaginationResult, error) {
	q := r.client.Supplier.Query()
	if status != "" {
		q = q.Where(dbsupplier.StatusEQ(status))
	}
	if search != "" {
		q = q.Where(dbsupplier.Or(dbsupplier.NameContainsFold(search), dbsupplier.BaseURLContainsFold(search)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, nil, err
	}
	models, err := q.Order(dbent.Desc(dbsupplier.FieldID)).Offset(params.Offset()).Limit(params.Limit()).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	ids := make([]int64, 0, len(models))
	for _, m := range models {
		ids = append(ids, m.ID)
	}
	counts := make(map[int64]int64)
	if len(ids) > 0 {
		var grouped []struct {
			SupplierID int64 `json:"supplier_id"`
			Count      int64 `json:"count"`
		}
		err = r.client.Account.Query().Where(dbaccount.SupplierIDIn(ids...)).GroupBy(dbaccount.FieldSupplierID).Aggregate(dbent.Count()).Scan(ctx, &grouped)
		if err != nil {
			return nil, nil, err
		}
		for _, item := range grouped {
			counts[item.SupplierID] = item.Count
		}
	}
	out := make([]service.Supplier, 0, len(models))
	for _, m := range models {
		s := supplierEntityToService(m)
		s.AccountCount = counts[m.ID]
		out = append(out, *s)
	}
	return out, paginationResultFromTotal(int64(total), params), nil
}

func (r *supplierRepository) Update(ctx context.Context, in *service.Supplier, expectedRevision int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	current, err := client.Supplier.Query().Where(dbsupplier.IDEQ(in.ID)).ForUpdate().Only(ctx)
	if err != nil {
		return translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	if current.Revision != expectedRevision {
		return service.ErrSupplierChanged
	}
	if in.BalanceAccountID != nil {
		ok, err := client.Account.Query().Where(dbaccount.IDEQ(*in.BalanceAccountID), dbaccount.SupplierIDEQ(in.ID)).Exist(ctx)
		if err != nil {
			return err
		}
		if !ok {
			return service.ErrAccountNotFound
		}
	}
	update := client.Supplier.UpdateOne(current).SetName(in.Name).SetBaseURL(in.BaseURL).SetStatus(in.Status).
		SetNotes(in.Notes).SetBalanceAPIKey(in.BalanceAPIKey).AddRevision(1)
	if in.BalanceAccountID == nil {
		update.ClearBalanceAccountID()
	} else {
		update.SetBalanceAccountID(*in.BalanceAccountID)
	}
	if current.BaseURL != in.BaseURL || current.BalanceAPIKey != in.BalanceAPIKey || !sameSupplierAccountID(current.BalanceAccountID, in.BalanceAccountID) {
		update.ClearBalance().ClearBalanceUpdatedAt().ClearBalanceCheckedAt().SetBalanceStatus("unknown").SetBalanceError("")
	}
	updated, err := update.Save(ctx)
	if err != nil {
		return err
	}
	accountIDs, err := client.Account.Query().Where(dbaccount.SupplierIDEQ(in.ID)).IDs(ctx)
	if err != nil {
		return err
	}
	if len(accountIDs) > 0 {
		_, err = client.ExecContext(ctx, `UPDATE accounts SET
			credentials = jsonb_set(COALESCE(credentials, '{}'::jsonb), '{base_url}', to_jsonb($2::text), true),
			supplier_disabled = $3,
			extra = CASE WHEN $4 THEN COALESCE(extra, '{}'::jsonb) - 'upstream_billing_probe' - 'ollama_cloud_usage_snapshot' - 'opencode_go_usage_snapshot' ELSE extra END,
			updated_at = NOW() WHERE supplier_id = $1 AND deleted_at IS NULL`, in.ID, in.BaseURL, in.Status != service.StatusActive, current.BaseURL != in.BaseURL)
		if err != nil {
			return err
		}
		if err := enqueueSchedulerOutbox(ctx, client, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, map[string]any{"account_ids": accountIDs}); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	*in = *supplierEntityToService(updated)
	in.AccountCount = int64(len(accountIDs))
	r.accounts.syncSchedulerAccountSnapshots(context.WithoutCancel(ctx), accountIDs)
	return nil
}

func (r *supplierRepository) Delete(ctx context.Context, id int64) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	client := tx.Client()
	if _, err := client.Supplier.Query().Where(dbsupplier.IDEQ(id)).ForUpdate().Only(ctx); err != nil {
		return translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	rows, err := client.QueryContext(ctx, `WITH RECURSIVE owned AS (
		SELECT id FROM accounts WHERE supplier_id = $1
		UNION SELECT a.id FROM accounts a JOIN owned p ON a.parent_account_id = p.id
	) SELECT id FROM owned ORDER BY id`, id)
	if err != nil {
		return err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var accountID int64
		if err := rows.Scan(&accountID); err != nil {
			_ = rows.Close()
			return err
		}
		ids = append(ids, accountID)
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	if len(ids) > 0 {
		groupRows, err := client.QueryContext(ctx, `SELECT DISTINCT group_id FROM account_groups WHERE account_id = ANY($1)`, pq.Array(ids))
		if err != nil {
			return err
		}
		groups := make([]int64, 0)
		for groupRows.Next() {
			var groupID int64
			if err := groupRows.Scan(&groupID); err != nil {
				_ = groupRows.Close()
				return err
			}
			groups = append(groups, groupID)
		}
		err = groupRows.Err()
		_ = groupRows.Close()
		if err != nil {
			return err
		}
		for _, query := range []string{
			`DELETE FROM account_groups WHERE account_id = ANY($1)`,
			`DELETE FROM scheduled_test_plans WHERE account_id = ANY($1)`,
			`UPDATE accounts SET deleted_at = COALESCE(deleted_at, NOW()), updated_at = NOW(), supplier_disabled = TRUE, credentials = '{}'::jsonb WHERE id = ANY($1)`,
			`UPDATE suppliers SET balance_account_id = NULL, balance = NULL, balance_status = 'unknown', balance_error = '', balance_updated_at = NULL, balance_checked_at = NULL, revision = revision + 1 WHERE balance_account_id = ANY($1) AND deleted_at IS NULL`,
		} {
			if _, err := client.ExecContext(ctx, query, pq.Array(ids)); err != nil {
				return err
			}
		}
		if err := enqueueSchedulerOutbox(ctx, client, service.SchedulerOutboxEventAccountBulkChanged, nil, nil, map[string]any{"account_ids": ids, "group_ids": groups}); err != nil {
			return err
		}
	}
	_, err = client.Supplier.Update().Where(dbsupplier.IDEQ(id), dbsupplier.DeletedAtIsNil()).SetDeletedAt(time.Now()).SetBalanceAPIKey("").ClearBalanceAccountID().AddRevision(1).Save(ctx)
	if err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	for _, accountID := range ids {
		r.accounts.deleteSchedulerAccountSnapshot(context.WithoutCancel(ctx), accountID)
	}
	return nil
}

func (r *supplierRepository) BeginBalanceRefresh(ctx context.Context, id int64) (*service.Supplier, error) {
	m, err := r.client.Supplier.UpdateOneID(id).Where(dbsupplier.DeletedAtIsNil()).AddBalanceProbeSequence(1).Save(ctx)
	if err != nil {
		return nil, translatePersistenceError(err, service.ErrSupplierNotFound, nil)
	}
	return supplierEntityToService(m), nil
}

func (r *supplierRepository) SaveBalanceSnapshot(ctx context.Context, id, revision, sequence int64, balance *float64, status, message string, now time.Time) (bool, error) {
	result, err := r.sql.ExecContext(ctx, `UPDATE suppliers SET
		balance = CASE WHEN $4::numeric IS NOT NULL AND $5 = 'success' THEN $4::numeric ELSE balance END,
		balance_updated_at = CASE WHEN $4::numeric IS NOT NULL AND $5 = 'success' THEN $7 ELSE balance_updated_at END,
		balance_status = $5, balance_error = $6, balance_checked_at = $7
		WHERE id = $1 AND revision = $2 AND balance_probe_sequence = $3 AND deleted_at IS NULL`,
		id, revision, sequence, balance, status, message, now)
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n == 1, err
}

// Supplier row locks precede account locks, so URL changes and cascade deletion
// cannot race a newly created or edited supplier account.
func prepareAccountSupplier(ctx context.Context, client *dbent.Client, account *service.Account) error {
	if account.SupplierID == nil && account.ParentAccountID != nil {
		parent, err := client.Account.Get(ctx, *account.ParentAccountID)
		if err != nil {
			return translatePersistenceError(err, service.ErrAccountNotFound, nil)
		}
		account.SupplierID = parent.SupplierID
	}
	var desiredID int64
	if account.SupplierID != nil {
		desiredID = *account.SupplierID
	}
	rows, err := client.QueryContext(ctx, `SELECT id FROM suppliers
		WHERE deleted_at IS NULL AND (id = $1 OR id = (SELECT supplier_id FROM accounts WHERE id = $2) OR balance_account_id = $2)
		ORDER BY id FOR UPDATE`, desiredID, account.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			_ = rows.Close()
			return err
		}
	}
	err = rows.Err()
	_ = rows.Close()
	if err != nil {
		return err
	}
	account.SupplierDisabled = false
	account.SupplierName, account.SupplierBaseURL = "", ""
	if desiredID > 0 {
		supplier, err := client.Supplier.Get(ctx, desiredID)
		if err != nil {
			return translatePersistenceError(err, service.ErrSupplierNotFound, nil)
		}
		account.SupplierName, account.SupplierBaseURL = supplier.Name, supplier.BaseURL
		account.SupplierDisabled = supplier.Status != service.StatusActive
		account.Credentials = copyJSONMap(account.Credentials)
		if account.Credentials == nil {
			account.Credentials = make(map[string]any)
		}
		account.Credentials["base_url"] = supplier.BaseURL
	}
	return nil
}

// In automatic mode any owned account may supply the wallet response. Changes
// to any candidate must reject requests started with the old candidate set.
const supplierBalanceSourceAccountSQL = `(s.balance_account_id = a.id OR
	(s.balance_account_id IS NULL AND s.balance_api_key = '' AND s.id = a.supplier_id))`

func invalidateSupplierBalanceForAccount(ctx context.Context, client *dbent.Client, accountID int64, deleted bool) error {
	_, err := client.ExecContext(ctx, `UPDATE suppliers s SET revision = revision + 1,
		balance = NULL, balance_status = 'unknown', balance_error = '', balance_updated_at = NULL, balance_checked_at = NULL,
		balance_account_id = CASE WHEN $2 THEN NULL ELSE balance_account_id END
		WHERE s.deleted_at IS NULL AND EXISTS (SELECT 1 FROM accounts a
			WHERE a.id = $1 AND a.deleted_at IS NULL AND `+supplierBalanceSourceAccountSQL+`)`, accountID, deleted)
	return err
}

func invalidateSupplierBalanceForAccountUpdate(ctx context.Context, client *dbent.Client, account *service.Account) error {
	credentials, err := json.Marshal(normalizeJSONMap(account.Credentials))
	if err != nil {
		return err
	}
	_, err = client.ExecContext(ctx, `UPDATE suppliers s SET revision = revision + 1,
		balance = NULL, balance_status = 'unknown', balance_error = '', balance_updated_at = NULL, balance_checked_at = NULL,
		balance_account_id = CASE WHEN s.id IS DISTINCT FROM $3::bigint THEN NULL ELSE s.balance_account_id END
		WHERE s.deleted_at IS NULL AND EXISTS (
			SELECT 1 FROM accounts a WHERE a.id = $1 AND a.deleted_at IS NULL
			AND (s.balance_account_id = a.id OR (s.balance_account_id IS NULL AND s.balance_api_key = ''
				AND (s.id = a.supplier_id OR s.id = $3::bigint))) AND (
				a.credentials -> 'api_key' IS DISTINCT FROM $2::jsonb -> 'api_key'
				OR a.credentials -> 'base_url' IS DISTINCT FROM $2::jsonb -> 'base_url'
				OR a.supplier_id IS DISTINCT FROM $3::bigint OR a.proxy_id IS DISTINCT FROM $4::bigint
				OR a.type IS DISTINCT FROM $5 OR a.platform IS DISTINCT FROM $6))`,
		account.ID, string(credentials), account.SupplierID, account.ProxyID, account.Type, account.Platform)
	return err
}

func sameSupplierAccountID(a, b *int64) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func (r *accountRepository) loadSuppliersForAccounts(ctx context.Context, accounts []*dbent.Account) (map[int64]*dbent.Supplier, error) {
	ids := make([]int64, 0)
	for _, account := range accounts {
		if account.SupplierID != nil {
			ids = append(ids, *account.SupplierID)
		}
	}
	result := make(map[int64]*dbent.Supplier)
	if len(ids) == 0 {
		return result, nil
	}
	models, err := r.client.Supplier.Query().Where(dbsupplier.IDIn(uniquePositiveInt64s(ids)...)).
		Select(dbsupplier.FieldID, dbsupplier.FieldName, dbsupplier.FieldBaseURL, dbsupplier.FieldStatus).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, model := range models {
		result[model.ID] = model
	}
	return result, nil
}

func applyAccountSupplier(account *service.Account, suppliers map[int64]*dbent.Supplier) {
	if account.SupplierID == nil {
		return
	}
	supplier, ok := suppliers[*account.SupplierID]
	if !ok {
		account.SupplierDisabled = true
		return
	}
	account.SupplierName, account.SupplierBaseURL = supplier.Name, supplier.BaseURL
	account.SupplierDisabled = supplier.Status != service.StatusActive
}

func supplierAccountFilter(ctx context.Context, q *dbent.AccountQuery) *dbent.AccountQuery {
	if id, ok := service.SupplierAccountFilterFromContext(ctx); ok {
		if id == 0 {
			return q.Where(dbaccount.SupplierIDIsNil())
		}
		return q.Where(dbaccount.SupplierIDEQ(id))
	}
	return q
}

var _ service.SupplierRepository = (*supplierRepository)(nil)
