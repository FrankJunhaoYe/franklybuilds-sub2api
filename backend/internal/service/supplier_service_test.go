package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
	"github.com/stretchr/testify/require"
)

func TestSupplierWalletBalanceDoesNotConfuseQuotaOrSubscription(t *testing.T) {
	for _, test := range []struct {
		name, body, status string
		balance            *float64
	}{
		{"wallet", `{"mode":"unrestricted","unit":"USD","balance":12.5,"remaining":99}`, "success", supplierFloat(12.5)},
		{"zero_wallet", `{"mode":"unrestricted","unit":"USD","balance":0}`, "success", supplierFloat(0)},
		{"overdrawn_wallet", `{"mode":"unrestricted","unit":"USD","balance":-0.1}`, "success", supplierFloat(-0.1)},
		{"key_quota", `{"mode":"quota_limited","unit":"USD","remaining":90}`, "unavailable", nil},
		{"subscription", `{"mode":"unrestricted","unit":"USD","remaining":90,"subscription":{"daily_limit_usd":100}}`, "unavailable", nil},
		{"misleading_subscription_balance", `{"mode":"unrestricted","unit":"USD","balance":90,"subscription":{}}`, "unavailable", nil},
		{"missing_balance", `{"mode":"unrestricted","unit":"USD","remaining":-1}`, "unavailable", nil},
		{"wrong_currency", `{"mode":"unrestricted","unit":"CNY","balance":12}`, "error", nil},
		{"invalid_response", `<html>login required</html>`, "error", nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			balance, status, _ := parseSupplierWalletBalance([]byte(test.body))
			require.Equal(t, test.status, status)
			require.Equal(t, test.balance, balance)
		})
	}
}

func supplierFloat(value float64) *float64 { return &value }
func supplierString(value string) *string  { return &value }

type supplierServiceRepoStub struct {
	SupplierRepository
	value  Supplier
	accept bool
}

func (r *supplierServiceRepoStub) GetByID(context.Context, int64) (*Supplier, error) {
	copy := r.value
	return &copy, nil
}
func (r *supplierServiceRepoStub) BeginBalanceRefresh(context.Context, int64) (*Supplier, error) {
	r.value.BalanceProbeSequence++
	copy := r.value
	return &copy, nil
}
func (r *supplierServiceRepoStub) SaveBalanceSnapshot(_ context.Context, _ int64, revision, sequence int64, balance *float64, status, code string, at time.Time) (bool, error) {
	if !r.accept || revision != r.value.Revision || sequence != r.value.BalanceProbeSequence {
		return false, nil
	}
	r.value.BalanceStatus, r.value.BalanceError, r.value.BalanceCheckedAt = status, code, &at
	if status == "success" {
		r.value.Balance, r.value.BalanceUpdatedAt = balance, &at
	}
	return true, nil
}
func (r *supplierServiceRepoStub) Update(_ context.Context, value *Supplier, expected int64) error {
	if expected != r.value.Revision {
		return ErrSupplierChanged
	}
	r.value = *value
	r.value.Revision++
	return nil
}

type supplierHTTPStub struct {
	handler func(*http.Request) (*http.Response, error)
}

func (s supplierHTTPStub) Do(r *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
	return s.handler(r)
}
func (s supplierHTTPStub) DoWithTLS(r *http.Request, proxy string, id int64, concurrency int, _ *tlsfingerprint.Profile) (*http.Response, error) {
	return s.Do(r, proxy, id, concurrency)
}

func TestSupplierRefreshUsesDedicatedCredentialAndPreservesLastSuccess(t *testing.T) {
	previous := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 3, BaseURL: "https://supplier.example/v1", BalanceAPIKey: "test-only-secret", Balance: supplierFloat(18), BalanceUpdatedAt: &previous}}
	httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "https://supplier.example/v1/usage", req.URL.String())
		require.Equal(t, "Bearer test-only-secret", req.Header.Get("Authorization"))
		require.True(t, HTTPUpstreamRedirectsDisabled(req.Context()))
		_, hasDeadline := req.Context().Deadline()
		require.True(t, hasDeadline)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"mode":"quota_limited","remaining":500}`))}, nil
	}}
	testService := &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}
	s := NewSupplierService(repo, nil, testService, nil)
	result, err := s.RefreshBalance(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "unavailable", result.BalanceStatus)
	require.Equal(t, supplierFloat(18), result.Balance)
	require.Equal(t, &previous, result.BalanceUpdatedAt)
	payload, err := json.Marshal(result)
	require.NoError(t, err)
	require.NotContains(t, string(payload), "test-only-secret")
	require.NotContains(t, string(payload), `"balance_api_key"`)
}

func TestSupplierRefreshRejectsChangedConfiguration(t *testing.T) {
	repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 3, BaseURL: "https://supplier.example", BalanceAPIKey: "old-key"}}
	httpClient := supplierHTTPStub{handler: func(*http.Request) (*http.Response, error) {
		repo.value.Revision++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","balance":300}`))}, nil
	}}
	s := NewSupplierService(repo, nil, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
	_, err := s.RefreshBalance(context.Background(), 7)
	require.ErrorIs(t, err, ErrSupplierChanged)
	require.Nil(t, repo.value.Balance)
}

func TestSupplierUpdateCredentialSemanticsAndInvalidation(t *testing.T) {
	repo := &supplierServiceRepoStub{value: Supplier{ID: 7, Name: "source", BaseURL: "https://supplier.example", Status: StatusActive, BalanceAPIKey: "stored-secret", Balance: supplierFloat(15), BalanceStatus: "success"}}
	s := NewSupplierService(repo, nil, nil, nil)
	result, err := s.Update(context.Background(), 7, SupplierInput{Notes: supplierString("memo")})
	require.NoError(t, err)
	require.Equal(t, "stored-secret", result.BalanceAPIKey)
	require.Equal(t, supplierFloat(15), result.Balance)
	result, err = s.Update(context.Background(), 7, SupplierInput{BalanceAPIKey: supplierString("")})
	require.NoError(t, err)
	require.Empty(t, result.BalanceAPIKey)
	require.False(t, result.HasBalanceAPIKey)
	require.Nil(t, result.Balance)
	require.Equal(t, "unknown", result.BalanceStatus)
}

func TestSupplierRejectsCredentialBearingURLs(t *testing.T) {
	s := NewSupplierService(nil, nil, nil, nil)
	for _, raw := range []string{"https://user:password@example.com", "https://example.com?api_key=secret", "https://example.com/#secret", "ftp://example.com", "https://"} {
		err := s.applyInput(context.Background(), &Supplier{Status: StatusActive}, SupplierInput{Name: supplierString("source"), BaseURL: &raw})
		require.Error(t, err, raw)
		require.NotContains(t, err.Error(), "password")
		require.NotContains(t, err.Error(), "secret")
	}
}

type supplierAccountReadStub struct {
	AccountRepository
	account Account
}

func (s supplierAccountReadStub) GetByID(context.Context, int64) (*Account, error) {
	copy := s.account
	return &copy, nil
}

func TestSupplierBalanceSourceMustBelongToSupplier(t *testing.T) {
	otherSupplierID, accountID := int64(10), int64(2)
	s := NewSupplierService(nil, supplierAccountReadStub{account: Account{ID: accountID, SupplierID: &otherSupplierID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}, nil, nil)
	err := s.applyInput(context.Background(), &Supplier{ID: 7, Name: "source", BaseURL: "https://supplier.example", Status: SupplierStatusActive}, SupplierInput{BalanceAccountID: &accountID, BalanceAccountIDSet: true})
	require.Error(t, err)
	require.True(t, infraerrors.IsBadRequest(err))
}

func TestSupplierAssignmentRejectsScopedAdministrator(t *testing.T) {
	supplierID := int64(7)
	ctx := adminaccess.WithPrincipal(context.Background(), &adminaccess.Principal{KeyID: "restricted"})
	admin := &adminServiceImpl{accountRepo: supplierAccountReadStub{account: Account{ID: 2, Platform: PlatformOpenAI, Type: AccountTypeAPIKey}}}
	_, err := admin.CreateAccount(ctx, &CreateAccountInput{SupplierID: &supplierID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey})
	require.True(t, infraerrors.IsForbidden(err))
	_, err = admin.UpdateAccount(ctx, 2, &UpdateAccountInput{SupplierIDSet: true, SupplierID: &supplierID})
	require.True(t, infraerrors.IsForbidden(err))
	_, err = admin.UpdateAccount(ctx, 2, &UpdateAccountInput{SupplierIDSet: true})
	require.True(t, infraerrors.IsForbidden(err), "detaching must not bypass a disabled supplier")
}

type supplierAccountWriteStub struct {
	supplierAccountReadStub
}

func (s *supplierAccountWriteStub) Create(_ context.Context, account *Account) error {
	account.ID = 2
	s.account = *account
	return nil
}

func (s *supplierAccountWriteStub) Update(_ context.Context, account *Account) error {
	s.account = *account
	return nil
}

func TestSupplierRestrictionsPreserveOrdinaryScopedAccountOperations(t *testing.T) {
	ctx := adminaccess.WithPrincipal(context.Background(), &adminaccess.Principal{KeyID: "restricted"})
	repo := &supplierAccountWriteStub{}
	admin := &adminServiceImpl{accountRepo: repo}
	created, err := admin.CreateAccount(ctx, &CreateAccountInput{Name: "own", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, SkipDefaultGroupBind: true})
	require.NoError(t, err)
	require.Nil(t, created.SupplierID)
	for _, owned := range []bool{false, true} {
		if owned {
			supplierID := int64(7)
			repo.account.SupplierID = &supplierID
		}
		updated, err := admin.UpdateAccount(ctx, 2, &UpdateAccountInput{Name: "renamed"})
		require.NoError(t, err, "ordinary account updates keep existing permissions")
		require.Equal(t, "renamed", updated.Name)
		require.Equal(t, repo.account.SupplierID, updated.SupplierID)
	}
}

func TestSupplierAccountSaveRefreshWaitsForSuccessfulCommit(t *testing.T) {
	for _, mode := range []string{"commit", "rollback", "commit_failure"} {
		t.Run(mode, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			require.NoError(t, err)
			defer func() { _ = db.Close() }()
			client := dbent.NewClient(dbent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			mock.ExpectBegin()
			tx, err := client.Tx(context.Background())
			require.NoError(t, err)
			repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example", BalanceAPIKey: "test-only"}}
			calls := 0
			httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
				require.Nil(t, dbent.TxFromContext(req.Context()), "probe must not reuse a committed transaction")
				calls++
				return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","balance":12}`))}, nil
			}}
			s := NewSupplierService(repo, nil, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
			supplierID := int64(7)
			s.RefreshAfterAccountSave(dbent.NewTxContext(context.Background(), tx), &Account{ID: 2, SupplierID: &supplierID})
			require.Zero(t, calls)
			require.Zero(t, repo.value.BalanceProbeSequence, "refresh cannot acquire supplier locks before commit")
			switch mode {
			case "commit":
				mock.ExpectCommit()
				require.NoError(t, tx.Commit())
				require.Equal(t, 1, calls)
				require.Equal(t, "success", repo.value.BalanceStatus)
			case "rollback":
				mock.ExpectRollback()
				require.NoError(t, tx.Rollback())
				require.Zero(t, calls)
			case "commit_failure":
				mock.ExpectCommit().WillReturnError(errors.New("commit failed"))
				require.Error(t, tx.Commit())
				require.Zero(t, calls)
			}
			require.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

type supplierAutoAccountsStub struct {
	AccountRepository
	accounts            []Account
	listCalls, getCalls int
	filteredSupplierID  int64
	filteredType        string
}

func (s *supplierAutoAccountsStub) ListAllWithFilters(ctx context.Context, _ string, accountType, _, _ string, _ int64, _ string) ([]Account, error) {
	s.listCalls++
	s.filteredSupplierID, _ = SupplierAccountFilterFromContext(ctx)
	s.filteredType = accountType
	return append([]Account(nil), s.accounts...), nil
}

func (s *supplierAutoAccountsStub) GetByID(_ context.Context, id int64) (*Account, error) {
	s.getCalls++
	for i := range s.accounts {
		if s.accounts[i].ID == id {
			copy := s.accounts[i]
			return &copy, nil
		}
	}
	return nil, ErrAccountNotFound
}

func supplierAutomaticAccount(id, supplierID int64, key string) Account {
	return Account{ID: id, SupplierID: &supplierID, Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": key}, Concurrency: 1}
}

func TestSupplierAccountSaveAutomaticallyUsesSavedKey(t *testing.T) {
	for _, hasOtherAccounts := range []bool{false, true} {
		accounts := &supplierAutoAccountsStub{accounts: []Account{supplierAutomaticAccount(1, 7, "saved-key")}}
		if hasOtherAccounts {
			accounts.accounts = append(accounts.accounts, supplierAutomaticAccount(5, 7, "newer-key"))
		}
		repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example"}}
		calls := 0
		httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
			calls++
			require.Equal(t, "Bearer saved-key", req.Header.Get("Authorization"))
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","balance":12.5}`))}, nil
		}}
		s := NewSupplierService(repo, accounts, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
		s.RefreshAfterAccountSave(context.Background(), &accounts.accounts[0])
		require.Equal(t, 1, calls)
		require.Equal(t, int64(7), accounts.filteredSupplierID)
		require.Equal(t, AccountTypeAPIKey, accounts.filteredType)
		require.Equal(t, "success", repo.value.BalanceStatus)
		require.Equal(t, supplierFloat(12.5), repo.value.Balance)
		require.Nil(t, repo.value.BalanceAccountID, "automatic selection must stay automatic")
		require.Empty(t, repo.value.BalanceAPIKey)
	}
}

func TestSupplierAutomaticBalanceFallsBackWithinSupplierAndStopsAtFirstWallet(t *testing.T) {
	accounts := &supplierAutoAccountsStub{accounts: []Account{
		supplierAutomaticAccount(1, 7, "unused-key"), supplierAutomaticAccount(2, 7, "working-key"),
		supplierAutomaticAccount(3, 7, "expired-key"), supplierAutomaticAccount(99, 8, "other-supplier"),
		supplierAutomaticAccount(100, 7, ""),
	}}
	wrongType := supplierAutomaticAccount(101, 7, "wrong-type")
	wrongType.Type = AccountTypeOAuth
	accounts.accounts = append(accounts.accounts, wrongType)
	repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example"}}
	var keys []string
	var firstDeadline time.Time
	httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
		key := req.Header.Get("Authorization")
		keys = append(keys, key)
		deadline, ok := req.Context().Deadline()
		require.True(t, ok)
		if firstDeadline.IsZero() {
			firstDeadline = deadline
		} else {
			require.Equal(t, firstDeadline, deadline, "fallbacks share one total timeout")
		}
		if key == "Bearer expired-key" {
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","balance":27}`))}, nil
	}}
	s := NewSupplierService(repo, accounts, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
	result, err := s.RefreshBalance(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, []string{"Bearer expired-key", "Bearer working-key"}, keys)
	require.Equal(t, supplierFloat(27), result.Balance)
}

func TestSupplierAutomaticBalanceHasBoundedFailuresAndPreservesLastSuccess(t *testing.T) {
	for _, count := range []int{0, 5} {
		previous := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		accounts := &supplierAutoAccountsStub{}
		for i := 1; i <= count; i++ {
			accounts.accounts = append(accounts.accounts, supplierAutomaticAccount(int64(i), 7, "invalid-key"))
		}
		repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example", Balance: supplierFloat(19), BalanceUpdatedAt: &previous}}
		calls := 0
		httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
			calls++
			deadline, ok := req.Context().Deadline()
			require.True(t, ok)
			require.LessOrEqual(t, time.Until(deadline), upstreamBillingProbeRequestTimeout)
			return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","remaining":999}`))}, nil
		}}
		s := NewSupplierService(repo, accounts, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
		result, err := s.RefreshBalance(context.Background(), 7)
		require.NoError(t, err)
		require.Equal(t, min(count, supplierBalanceAutoMaxAttempts), calls)
		require.Equal(t, "unavailable", result.BalanceStatus)
		require.Equal(t, supplierFloat(19), result.Balance)
		require.Equal(t, &previous, result.BalanceUpdatedAt)
	}
}

func TestSupplierExplicitBalanceSourcesDoNotFallBackToOtherKeys(t *testing.T) {
	for _, dedicated := range []bool{false, true} {
		accounts := &supplierAutoAccountsStub{accounts: []Account{supplierAutomaticAccount(1, 7, "selected-key"), supplierAutomaticAccount(2, 7, "other-key")}}
		repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example"}}
		expectedKey := "selected-key"
		if dedicated {
			expectedKey = "dedicated-key"
			repo.value.BalanceAPIKey = expectedKey
		} else {
			id := int64(1)
			repo.value.BalanceAccountID = &id
		}
		calls := 0
		httpClient := supplierHTTPStub{handler: func(req *http.Request) (*http.Response, error) {
			calls++
			require.Equal(t, "Bearer "+expectedKey, req.Header.Get("Authorization"))
			return &http.Response{StatusCode: http.StatusUnauthorized, Body: io.NopCloser(strings.NewReader(`{}`))}, nil
		}}
		s := NewSupplierService(repo, accounts, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
		result, err := s.RefreshBalance(context.Background(), 7)
		require.NoError(t, err)
		require.Equal(t, 1, calls)
		require.Zero(t, accounts.listCalls)
		require.Equal(t, "error", result.BalanceStatus)
	}
}

func TestSupplierAutomaticBalanceRejectsChangedAccountConfiguration(t *testing.T) {
	accounts := &supplierAutoAccountsStub{accounts: []Account{supplierAutomaticAccount(1, 7, "old-key")}}
	repo := &supplierServiceRepoStub{accept: true, value: Supplier{ID: 7, Revision: 1, BaseURL: "https://supplier.example"}}
	httpClient := supplierHTTPStub{handler: func(*http.Request) (*http.Response, error) {
		repo.value.Revision++
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"mode":"unrestricted","unit":"USD","balance":300}`))}, nil
	}}
	s := NewSupplierService(repo, accounts, &AccountTestService{cfg: &config.Config{}, httpUpstream: httpClient}, nil)
	_, err := s.RefreshBalance(context.Background(), 7)
	require.ErrorIs(t, err, ErrSupplierChanged)
	require.Nil(t, repo.value.Balance)
}

func TestSupplierAccountCreationDefaultsAndExplicitSettings(t *testing.T) {
	for _, test := range []struct {
		name                  string
		own                   bool
		configure             func(*CreateAccountInput)
		concurrency, priority int
		pool, probe, rateSync bool
		wantError             bool
	}{
		{name: "supplier_defaults", concurrency: 100, priority: 1, pool: true, probe: true, rateSync: true},
		{name: "own_defaults_unchanged", own: true},
		{name: "explicit_zero_and_false", configure: func(input *CreateAccountInput) {
			input.ConcurrencySet, input.PrioritySet = true, true
			disabled := false
			input.RateSyncEnabled = &disabled
			input.Credentials["pool_mode"] = false
		}},
		{name: "explicit_values", concurrency: 8, priority: 4, pool: true, probe: true, rateSync: true, configure: func(input *CreateAccountInput) {
			input.Concurrency, input.Priority = 8, 4
		}},
		{name: "explicit_probe_disabled", concurrency: 100, priority: 1, pool: true, configure: func(input *CreateAccountInput) {
			disabled := false
			input.ProbeEnabled = &disabled
		}},
		{name: "explicit_probe_sync_conflict", wantError: true, configure: func(input *CreateAccountInput) {
			disabled, enabled := false, true
			input.ProbeEnabled, input.RateSyncEnabled = &disabled, &enabled
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			input := &CreateAccountInput{Name: "key", Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Credentials: map[string]any{"api_key": "test-only"}, SkipDefaultGroupBind: true}
			if !test.own {
				id := int64(7)
				input.SupplierID = &id
			}
			if test.configure != nil {
				test.configure(input)
			}
			admin := &adminServiceImpl{accountRepo: &supplierAccountWriteStub{}}
			account, err := admin.CreateAccount(context.Background(), input)
			if test.wantError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, test.concurrency, account.Concurrency)
			require.Equal(t, test.priority, account.Priority)
			require.Equal(t, test.pool, account.IsPoolMode())
			require.Equal(t, test.probe, upstreamBillingProbeEnabled(account))
			require.Equal(t, test.rateSync, upstreamBillingRateSyncEnabled(account))
		})
	}
}
