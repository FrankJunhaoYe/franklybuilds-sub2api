package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

const (
	SupplierStatusActive   = "active"
	SupplierStatusInactive = "inactive"
)

var (
	ErrSupplierNotFound = infraerrors.NotFound("SUPPLIER_NOT_FOUND", "supplier not found")
	ErrSupplierChanged  = infraerrors.Conflict("SUPPLIER_CHANGED", "supplier configuration changed; retry the operation")
)

// Supplier is an administrator-managed commercial source of upstream accounts.
// The dedicated balance credential and concurrency tokens never leave the backend.
type Supplier struct {
	ID                   int64      `json:"id"`
	Name                 string     `json:"name"`
	BaseURL              string     `json:"base_url"`
	Status               string     `json:"status"`
	Notes                string     `json:"notes"`
	AccountCount         int64      `json:"account_count"`
	BalanceAccountID     *int64     `json:"balance_account_id"`
	BalanceAPIKey        string     `json:"-"`
	HasBalanceAPIKey     bool       `json:"has_balance_api_key"`
	Balance              *float64   `json:"balance"`
	BalanceStatus        string     `json:"balance_status"`
	BalanceError         string     `json:"balance_error"`
	BalanceUpdatedAt     *time.Time `json:"balance_updated_at"`
	BalanceCheckedAt     *time.Time `json:"balance_checked_at"`
	Revision             int64      `json:"-"`
	BalanceProbeSequence int64      `json:"-"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type SupplierRepository interface {
	Create(context.Context, *Supplier) error
	GetByID(context.Context, int64) (*Supplier, error)
	List(context.Context, pagination.PaginationParams, string, string) ([]Supplier, *pagination.PaginationResult, error)
	Update(context.Context, *Supplier, int64) error
	Delete(context.Context, int64) error
	// BeginBalanceRefresh serializes refresh starts and returns their configuration revision.
	BeginBalanceRefresh(context.Context, int64) (*Supplier, error)
	// SaveBalanceSnapshot only accepts the newest refresh of unchanged configuration.
	// Failure states preserve the previous successful balance and its timestamp.
	SaveBalanceSnapshot(context.Context, int64, int64, int64, *float64, string, string, time.Time) (bool, error)
}

type supplierAccountFilterContextKey struct{}

// WithSupplierAccountFilter filters account management reads; zero means self-managed accounts.
func WithSupplierAccountFilter(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, supplierAccountFilterContextKey{}, id)
}

func SupplierAccountFilterFromContext(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(supplierAccountFilterContextKey{}).(int64)
	return id, ok
}
