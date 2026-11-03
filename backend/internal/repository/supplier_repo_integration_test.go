//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func supplierRepositoryFixture(t *testing.T) (*supplierRepository, *service.Supplier) {
	t.Helper()
	client := testEntClient(t)
	repo := &supplierRepository{client: client, sql: integrationDB, accounts: newAccountRepositoryWithSQL(client, integrationDB, nil)}
	supplier := &service.Supplier{Name: fmt.Sprintf("supplier-%d", time.Now().UnixNano()), BaseURL: "https://supplier.example", Status: "active", BalanceAPIKey: "fixture-query-key"}
	require.NoError(t, repo.Create(context.Background(), supplier))
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DELETE FROM scheduler_outbox o WHERE EXISTS (
			SELECT 1 FROM accounts a WHERE a.supplier_id = $1 AND (
				o.account_id = a.id OR o.payload @> jsonb_build_object('account_ids', jsonb_build_array(a.id))))`, supplier.ID)
		_, _ = integrationDB.Exec(`UPDATE suppliers SET balance_account_id = NULL WHERE id = $1`, supplier.ID)
		_, _ = integrationDB.Exec(`DELETE FROM account_groups WHERE account_id IN (SELECT id FROM accounts WHERE supplier_id = $1)`, supplier.ID)
		_, _ = integrationDB.Exec(`DELETE FROM accounts WHERE supplier_id = $1`, supplier.ID)
		_, _ = integrationDB.Exec(`DELETE FROM suppliers WHERE id = $1`, supplier.ID)
	})
	return repo, supplier
}

func TestSupplierRepositoryLifecycle(t *testing.T) {
	ctx := context.Background()
	repo, supplier := supplierRepositoryFixture(t)
	client := repo.client
	group, err := client.Group.Create().SetName(supplier.Name).SetPlatform(service.PlatformOpenAI).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DELETE FROM account_groups WHERE group_id = $1`, group.ID)
		_, _ = integrationDB.Exec(`DELETE FROM groups WHERE id = $1`, group.ID)
	})
	account := &service.Account{Name: supplier.Name, SupplierID: &supplier.ID, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Credentials: map[string]any{"api_key": "fixture-key", "base_url": "https://wrong.example"}}
	require.NoError(t, repo.accounts.CreateWithAccountGroups(ctx, account, []service.AccountGroup{{GroupID: group.ID, Priority: 2}}))
	paused := &service.Account{Name: supplier.Name + "-paused", SupplierID: &supplier.ID, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: false, Credentials: map[string]any{"api_key": "paused-key"}}
	require.NoError(t, repo.accounts.CreateWithAccountGroups(ctx, paused, []service.AccountGroup{{GroupID: group.ID, Priority: 3}}))
	selfManaged := &service.Account{Name: supplier.Name + "-self", Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey,
		Status: service.StatusActive, Schedulable: true, Credentials: map[string]any{"api_key": "self-key"}}
	require.NoError(t, repo.accounts.CreateWithAccountGroups(ctx, selfManaged, []service.AccountGroup{{GroupID: group.ID, Priority: 4}}))
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DELETE FROM account_groups WHERE account_id = $1`, selfManaged.ID)
		_, _ = integrationDB.Exec(`DELETE FROM accounts WHERE id = $1`, selfManaged.ID)
	})
	loaded, err := repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, supplier.BaseURL, loaded.GetOpenAIBaseURL())
	list, page, err := repo.accounts.ListWithFilters(service.WithSupplierAccountFilter(ctx, supplier.ID), pagination.DefaultPagination(), "", "", "", "", 0, "")
	require.NoError(t, err)
	require.EqualValues(t, 2, page.Total)
	require.Len(t, list, 2)
	supplier.Status = "inactive"
	require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
	loaded, err = repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.False(t, loaded.IsSchedulable())
	require.Equal(t, service.StatusActive, loaded.Status)
	require.True(t, loaded.Schedulable)
	pool, err := repo.accounts.ListSchedulableByGroupID(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, pool, 1)
	require.Equal(t, selfManaged.ID, pool[0].ID)
	supplier.Status = "active"
	supplier.BaseURL = "https://replacement.example"
	require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
	loaded, err = repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, supplier.BaseURL, loaded.GetOpenAIBaseURL())
	require.True(t, loaded.IsSchedulable())
	stillPaused, err := repo.accounts.GetByID(ctx, paused.ID)
	require.NoError(t, err)
	require.False(t, stillPaused.IsSchedulable(), "enabling a supplier must preserve its account's manual pause")
	_, err = repo.accounts.BulkUpdate(ctx, []int64{account.ID}, service.AccountBulkUpdate{Credentials: map[string]any{"base_url": "https://bypass.example"}})
	require.NoError(t, err)
	loaded, err = repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, supplier.BaseURL, loaded.GetOpenAIBaseURL())

	user := mustCreateUser(t, client, &service.User{Email: fmt.Sprintf("%s@example.com", supplier.Name)})
	key := mustCreateApiKey(t, client, &service.APIKey{UserID: user.ID, Key: supplier.Name, Name: "history"})
	log, err := client.UsageLog.Create().SetUserID(user.ID).SetAPIKeyID(key.ID).SetAccountID(account.ID).SetRequestID(supplier.Name).SetModel("test-model").SetActualCost(1.25).Save(ctx)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(`DELETE FROM usage_logs WHERE id = $1`, log.ID)
		_, _ = integrationDB.Exec(`DELETE FROM api_keys WHERE id = $1`, key.ID)
		_, _ = integrationDB.Exec(`DELETE FROM users WHERE id = $1`, user.ID)
	})
	require.NoError(t, repo.accounts.Delete(ctx, paused.ID))
	require.NoError(t, repo.Delete(ctx, supplier.ID))
	_, err = repo.accounts.GetByID(ctx, account.ID)
	require.ErrorIs(t, err, service.ErrAccountNotFound)
	var cost float64
	require.NoError(t, integrationDB.QueryRow(`SELECT actual_cost FROM usage_logs WHERE id = $1`, log.ID).Scan(&cost))
	require.Equal(t, 1.25, cost)
	var credentialJSON, queryKey string
	require.NoError(t, integrationDB.QueryRow(`SELECT credentials::text FROM accounts WHERE id = $1`, account.ID).Scan(&credentialJSON))
	require.JSONEq(t, `{}`, credentialJSON)
	require.NoError(t, integrationDB.QueryRow(`SELECT credentials::text FROM accounts WHERE id = $1`, paused.ID).Scan(&credentialJSON))
	require.JSONEq(t, `{}`, credentialJSON, "supplier deletion also clears credentials of already soft-deleted keys")
	require.NoError(t, integrationDB.QueryRow(`SELECT balance_api_key FROM suppliers WHERE id = $1`, supplier.ID).Scan(&queryKey))
	require.Empty(t, queryKey)
	pool, err = repo.accounts.ListSchedulableByGroupID(ctx, group.ID)
	require.NoError(t, err)
	require.Len(t, pool, 1)
	require.Equal(t, selfManaged.ID, pool[0].ID)
}

func TestSupplierSelectedBalanceAccountInvalidatesInflightQuery(t *testing.T) {
	ctx := context.Background()
	repo, supplier := supplierRepositoryFixture(t)
	account := &service.Account{Name: supplier.Name, SupplierID: &supplier.ID, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Credentials: map[string]any{"api_key": "before"}}
	require.NoError(t, repo.accounts.Create(ctx, account))
	supplier.BalanceAccountID, supplier.BalanceAPIKey = &account.ID, ""
	require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
	first, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
	require.NoError(t, err)
	balance := 14.0
	accepted, err := repo.SaveBalanceSnapshot(ctx, supplier.ID, first.Revision, first.BalanceProbeSequence, &balance, "success", "", time.Now())
	require.NoError(t, err)
	require.True(t, accepted)
	loaded, err := repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	loaded.Name = "only the label changed"
	require.NoError(t, repo.accounts.Update(ctx, loaded))
	snapshot, err := repo.GetByID(ctx, supplier.ID)
	require.NoError(t, err)
	require.Equal(t, balance, *snapshot.Balance)
	require.Equal(t, first.Revision, snapshot.Revision)
	request, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
	require.NoError(t, err)
	require.NoError(t, repo.accounts.UpdateCredentials(ctx, account.ID, map[string]any{"api_key": "after", "base_url": "https://bypass.example"}))
	accepted, err = repo.SaveBalanceSnapshot(ctx, supplier.ID, request.Revision, request.BalanceProbeSequence, &balance, "success", "", time.Now())
	require.NoError(t, err)
	require.False(t, accepted)
	loaded, err = repo.accounts.GetByID(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, supplier.BaseURL, loaded.GetOpenAIBaseURL())
	require.NoError(t, repo.accounts.Delete(ctx, account.ID))
	snapshot, err = repo.GetByID(ctx, supplier.ID)
	require.NoError(t, err)
	require.Nil(t, snapshot.BalanceAccountID)
	require.Nil(t, snapshot.Balance)
}

func TestSupplierBalanceCASAndUnrelatedEdits(t *testing.T) {
	ctx := context.Background()
	repo, supplier := supplierRepositoryFixture(t)
	first, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
	require.NoError(t, err)
	second, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
	require.NoError(t, err)
	balance := 12.5
	accepted, err := repo.SaveBalanceSnapshot(ctx, supplier.ID, second.Revision, second.BalanceProbeSequence, &balance, "success", "", time.Now())
	require.NoError(t, err)
	require.True(t, accepted)
	old := 2.0
	accepted, err = repo.SaveBalanceSnapshot(ctx, supplier.ID, first.Revision, first.BalanceProbeSequence, &old, "success", "", time.Now())
	require.NoError(t, err)
	require.False(t, accepted)
	// This struct predates the successful query and contains a nil Balance.
	supplier.Notes = "updated while query was in flight"
	require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
	require.NotNil(t, supplier.Balance)
	require.Equal(t, balance, *supplier.Balance)
	third, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
	require.NoError(t, err)
	accepted, err = repo.SaveBalanceSnapshot(ctx, supplier.ID, third.Revision, third.BalanceProbeSequence, nil, "error", "query failed", time.Now())
	require.NoError(t, err)
	require.True(t, accepted)
	loaded, err := repo.GetByID(ctx, supplier.ID)
	require.NoError(t, err)
	require.Equal(t, balance, *loaded.Balance)
	require.Equal(t, "error", loaded.BalanceStatus)
	supplier.BaseURL = "https://new-wallet.example"
	require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
	accepted, err = repo.SaveBalanceSnapshot(ctx, supplier.ID, third.Revision, third.BalanceProbeSequence, &old, "success", "", time.Now())
	require.NoError(t, err)
	require.False(t, accepted)
	require.Nil(t, supplier.Balance)
}

func TestSupplierDeleteRollsBackAllAccounts(t *testing.T) {
	ctx := context.Background()
	repo, supplier := supplierRepositoryFixture(t)
	ids := make([]int64, 0, 2)
	for i := 0; i < 2; i++ {
		account := &service.Account{Name: fmt.Sprintf("%s-%d", supplier.Name, i), SupplierID: &supplier.ID, Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive, Schedulable: true, Credentials: map[string]any{"api_key": "fixture"}}
		require.NoError(t, repo.accounts.Create(ctx, account))
		ids = append(ids, account.ID)
	}
	functionName := fmt.Sprintf("supplier_delete_fail_%d", supplier.ID)
	_, err := integrationDB.Exec(fmt.Sprintf(`CREATE FUNCTION %s() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF OLD.id = %d AND NEW.deleted_at IS NOT NULL THEN RAISE EXCEPTION 'test failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER %s BEFORE UPDATE ON accounts FOR EACH ROW EXECUTE FUNCTION %s()`, functionName, ids[1], functionName, functionName))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = integrationDB.Exec(fmt.Sprintf(`DROP TRIGGER IF EXISTS %s ON accounts; DROP FUNCTION IF EXISTS %s()`, functionName, functionName))
	})
	require.Error(t, repo.Delete(ctx, supplier.ID))
	_, err = repo.GetByID(ctx, supplier.ID)
	require.NoError(t, err)
	for _, id := range ids {
		account, err := repo.accounts.GetByID(ctx, id)
		require.NoError(t, err)
		require.True(t, account.IsSchedulable())
	}
}

func TestSupplierAutomaticBalanceAccountChangesInvalidateInflightQuery(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name   string
		change func(*testing.T, *supplierRepository, *service.Account) error
	}{
		{"create", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			return repo.accounts.Create(ctx, &service.Account{Name: account.Name + "-new", SupplierID: account.SupplierID,
				Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive,
				Credentials: map[string]any{"api_key": "new-candidate"}})
		}},
		{"credentials", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			return repo.accounts.UpdateCredentials(ctx, account.ID, map[string]any{"api_key": "replacement"})
		}},
		{"account_credentials", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			account.Credentials["api_key"] = "replacement"
			return repo.accounts.Update(ctx, account)
		}},
		{"type", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			account.Type = service.AccountTypeOAuth
			return repo.accounts.Update(ctx, account)
		}},
		{"platform", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			account.Platform = service.PlatformAnthropic
			return repo.accounts.Update(ctx, account)
		}},
		{"bulk_credentials", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			_, err := repo.accounts.BulkUpdate(ctx, []int64{account.ID}, service.AccountBulkUpdate{Credentials: map[string]any{"api_key": "replacement"}})
			return err
		}},
		{"detach", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			account.SupplierID = nil
			return repo.accounts.Update(ctx, account)
		}},
		{"transfer", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			destinationRepo, destination := supplierRepositoryFixture(t)
			destination.BalanceAPIKey = ""
			require.NoError(t, destinationRepo.Update(ctx, destination, destination.Revision))
			request, err := destinationRepo.BeginBalanceRefresh(ctx, destination.ID)
			require.NoError(t, err)
			account.SupplierID = &destination.ID
			require.NoError(t, repo.accounts.Update(ctx, account))
			accepted, err := destinationRepo.SaveBalanceSnapshot(ctx, destination.ID, request.Revision, request.BalanceProbeSequence, nil, "unavailable", "", time.Now())
			require.NoError(t, err)
			require.False(t, accepted, "a new candidate also invalidates a request for the destination supplier")
			return nil
		}},
		{"delete", func(t *testing.T, repo *supplierRepository, account *service.Account) error {
			return repo.accounts.Delete(ctx, account.ID)
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo, supplier := supplierRepositoryFixture(t)
			supplier.BalanceAPIKey = ""
			require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
			account := &service.Account{Name: supplier.Name, SupplierID: &supplier.ID, Platform: service.PlatformOpenAI,
				Type: service.AccountTypeAPIKey, Status: service.StatusActive, Credentials: map[string]any{"api_key": "original"}}
			require.NoError(t, repo.accounts.Create(ctx, account))
			accountID := account.ID
			t.Cleanup(func() {
				_, _ = integrationDB.Exec(`DELETE FROM scheduler_outbox WHERE account_id = $1 OR payload @> jsonb_build_object('account_ids', jsonb_build_array($1::bigint))`, accountID)
				_, _ = integrationDB.Exec(`DELETE FROM accounts WHERE id = $1`, accountID)
			})
			account, err := repo.accounts.GetByID(ctx, accountID)
			require.NoError(t, err)
			assertSupplierBalanceChangeRejectsInflight(t, repo, supplier.ID, func() {
				require.NoError(t, tc.change(t, repo, account))
			})
		})
	}
}

func TestSupplierAutomaticBalanceProxyChangesInvalidateInflightQuery(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"account_proxy", "bulk_proxy", "proxy_identity", "expired_proxy", "expired_proxy_direct", "restore_proxy"} {
		t.Run(mode, func(t *testing.T) {
			repo, supplier := supplierRepositoryFixture(t)
			supplier.BalanceAPIKey = ""
			require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
			proxies := newProxyRepositoryWithSQL(repo.client, integrationDB)
			proxy := &service.Proxy{Name: supplier.Name, Protocol: "http", Host: "127.0.0.1", Port: 8080,
				Status: service.StatusActive, FallbackMode: service.FallbackModeNone}
			if mode == "expired_proxy" || mode == "expired_proxy_direct" {
				expired := time.Now().Add(-time.Hour)
				proxy.ExpiresAt = &expired
			}
			if mode == "expired_proxy_direct" {
				proxy.FallbackMode = service.FallbackModeDirect
			}
			require.NoError(t, proxies.Create(ctx, proxy))
			t.Cleanup(func() {
				_, _ = integrationDB.Exec(`UPDATE accounts SET proxy_id = NULL, proxy_fallback_origin_id = NULL WHERE proxy_id = $1 OR proxy_fallback_origin_id = $1`, proxy.ID)
				_, _ = integrationDB.Exec(`DELETE FROM proxies WHERE id = $1`, proxy.ID)
			})
			account := &service.Account{Name: supplier.Name, SupplierID: &supplier.ID, ProxyID: &proxy.ID,
				Platform: service.PlatformOpenAI, Type: service.AccountTypeAPIKey, Status: service.StatusActive,
				Credentials: map[string]any{"api_key": "original"}}
			require.NoError(t, repo.accounts.Create(ctx, account))
			if mode == "restore_proxy" {
				_, err := integrationDB.Exec(`UPDATE accounts SET proxy_id = NULL, proxy_fallback_origin_id = $1 WHERE id = $2`, proxy.ID, account.ID)
				require.NoError(t, err)
			}
			assertSupplierBalanceChangeRejectsInflight(t, repo, supplier.ID, func() {
				switch mode {
				case "account_proxy":
					account.ProxyID = nil
					require.NoError(t, repo.accounts.Update(ctx, account))
				case "bulk_proxy":
					var direct int64
					_, err := repo.accounts.BulkUpdate(ctx, []int64{account.ID}, service.AccountBulkUpdate{ProxyID: &direct})
					require.NoError(t, err)
				case "proxy_identity":
					proxy.Password = "changed-proxy-credential"
					require.NoError(t, proxies.Update(ctx, proxy))
				case "expired_proxy", "expired_proxy_direct":
					_, err := proxies.SweepExpiredProxies(ctx, time.Now())
					require.NoError(t, err)
				case "restore_proxy":
					require.NoError(t, repo.accounts.RevertProxyFallback(ctx, account.ID))
				}
			})
		})
	}
}

func TestSupplierBalanceInvalidationPreservesUnrelatedSources(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"dedicated_key", "selected_account", "automatic_name_only", "other_supplier"} {
		t.Run(mode, func(t *testing.T) {
			repo, supplier := supplierRepositoryFixture(t)
			account := &service.Account{Name: supplier.Name, SupplierID: &supplier.ID, Platform: service.PlatformOpenAI,
				Type: service.AccountTypeAPIKey, Status: service.StatusActive, Credentials: map[string]any{"api_key": "original"}}
			require.NoError(t, repo.accounts.Create(ctx, account))
			if mode != "dedicated_key" {
				supplier.BalanceAPIKey = ""
				if mode == "selected_account" {
					selected := &service.Account{Name: supplier.Name + "-selected", SupplierID: &supplier.ID, Platform: service.PlatformOpenAI,
						Type: service.AccountTypeAPIKey, Status: service.StatusActive, Credentials: map[string]any{"api_key": "selected"}}
					require.NoError(t, repo.accounts.Create(ctx, selected))
					supplier.BalanceAccountID = &selected.ID
				}
				require.NoError(t, repo.Update(ctx, supplier, supplier.Revision))
			}
			if mode == "other_supplier" {
				otherRepo, other := supplierRepositoryFixture(t)
				account.SupplierID, account.ID = &other.ID, 0
				require.NoError(t, otherRepo.accounts.Create(ctx, account))
			}
			request, err := repo.BeginBalanceRefresh(ctx, supplier.ID)
			require.NoError(t, err)
			if mode == "automatic_name_only" {
				account.Name = "renamed"
				require.NoError(t, repo.accounts.Update(ctx, account))
			} else {
				require.NoError(t, repo.accounts.UpdateCredentials(ctx, account.ID, map[string]any{"api_key": "changed"}))
			}
			balance := 16.5
			accepted, err := repo.SaveBalanceSnapshot(ctx, supplier.ID, request.Revision, request.BalanceProbeSequence, &balance, "success", "", time.Now())
			require.NoError(t, err)
			require.True(t, accepted, "an unrelated edit must preserve the current wallet request")
		})
	}
}

func assertSupplierBalanceChangeRejectsInflight(t *testing.T, repo *supplierRepository, supplierID int64, change func()) {
	t.Helper()
	ctx := context.Background()
	request, err := repo.BeginBalanceRefresh(ctx, supplierID)
	require.NoError(t, err)
	balance := 18.0
	accepted, err := repo.SaveBalanceSnapshot(ctx, supplierID, request.Revision, request.BalanceProbeSequence, &balance, "success", "", time.Now())
	require.NoError(t, err)
	require.True(t, accepted)
	request, err = repo.BeginBalanceRefresh(ctx, supplierID)
	require.NoError(t, err)
	change()
	accepted, err = repo.SaveBalanceSnapshot(ctx, supplierID, request.Revision, request.BalanceProbeSequence, &balance, "success", "", time.Now())
	require.NoError(t, err)
	require.False(t, accepted, "the previous candidate identity cannot repopulate the wallet")
	current, err := repo.GetByID(ctx, supplierID)
	require.NoError(t, err)
	require.Greater(t, current.Revision, request.Revision)
	require.Nil(t, current.Balance)
	require.Equal(t, "unknown", current.BalanceStatus)
}
