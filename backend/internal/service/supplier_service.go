package service

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/tlsfingerprint"
)

type SupplierInput struct {
	Name                *string
	BaseURL             *string
	Status              *string
	Notes               *string
	BalanceAccountID    *int64
	BalanceAccountIDSet bool
	BalanceAPIKey       *string
}

type SupplierService struct {
	repo     SupplierRepository
	accounts AccountRepository
	test     *AccountTestService
	billing  *UpstreamBillingProbeService
}

func NewSupplierService(repo SupplierRepository, accounts AccountRepository, test *AccountTestService, billing *UpstreamBillingProbeService) *SupplierService {
	return &SupplierService{repo: repo, accounts: accounts, test: test, billing: billing}
}

func (s *SupplierService) List(ctx context.Context, params pagination.PaginationParams, status, search string) ([]Supplier, *pagination.PaginationResult, error) {
	if status != "" && status != SupplierStatusActive && status != SupplierStatusInactive {
		return nil, nil, infraerrors.BadRequest("INVALID_SUPPLIER_STATUS", "invalid supplier status")
	}
	return s.repo.List(ctx, params, status, strings.TrimSpace(search))
}

func (s *SupplierService) GetByID(ctx context.Context, id int64) (*Supplier, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *SupplierService) Create(ctx context.Context, input SupplierInput) (*Supplier, error) {
	supplier := &Supplier{Status: SupplierStatusActive, BalanceStatus: "unknown"}
	if err := s.applyInput(ctx, supplier, input); err != nil {
		return nil, err
	}
	if err := s.repo.Create(ctx, supplier); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, supplier.ID)
}

func (s *SupplierService) Update(ctx context.Context, id int64, input SupplierInput) (*Supplier, error) {
	supplier, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	revision := supplier.Revision
	oldURL, oldKey, oldAccount := supplier.BaseURL, supplier.BalanceAPIKey, supplier.BalanceAccountID
	if err := s.applyInput(ctx, supplier, input); err != nil {
		return nil, err
	}
	if oldURL != supplier.BaseURL || oldKey != supplier.BalanceAPIKey || !sameSupplierAccountID(oldAccount, supplier.BalanceAccountID) {
		supplier.Balance = nil
		supplier.BalanceStatus = "unknown"
		supplier.BalanceError = ""
		supplier.BalanceUpdatedAt = nil
		supplier.BalanceCheckedAt = nil
	}
	if err := s.repo.Update(ctx, supplier, revision); err != nil {
		return nil, err
	}
	return s.repo.GetByID(ctx, id)
}

func sameSupplierAccountID(a, b *int64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func (s *SupplierService) applyInput(ctx context.Context, supplier *Supplier, input SupplierInput) error {
	if input.Name != nil {
		supplier.Name = strings.TrimSpace(*input.Name)
	}
	if supplier.Name == "" || len(supplier.Name) > 100 {
		return infraerrors.BadRequest("INVALID_SUPPLIER_NAME", "supplier name must contain 1-100 bytes")
	}
	if input.BaseURL != nil {
		base := strings.TrimRight(strings.TrimSpace(*input.BaseURL), "/")
		u, err := url.Parse(base)
		if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http") || len(base) > 2048 {
			return infraerrors.BadRequest("INVALID_SUPPLIER_URL", "supplier URL must be an HTTP(S) base URL without credentials, query or fragment")
		}
		if s.test != nil {
			if _, err := s.test.validateUpstreamBaseURL(base); err != nil {
				return infraerrors.BadRequest("INVALID_SUPPLIER_URL", "supplier URL is not allowed by the upstream URL policy")
			}
		}
		supplier.BaseURL = base
	}
	if supplier.BaseURL == "" {
		return infraerrors.BadRequest("INVALID_SUPPLIER_URL", "supplier URL is required")
	}
	if input.Status != nil {
		supplier.Status = *input.Status
	}
	if supplier.Status != SupplierStatusActive && supplier.Status != SupplierStatusInactive {
		return infraerrors.BadRequest("INVALID_SUPPLIER_STATUS", "invalid supplier status")
	}
	if input.Notes != nil {
		supplier.Notes = strings.TrimSpace(*input.Notes)
	}
	if len(supplier.Notes) > 10000 {
		return infraerrors.BadRequest("INVALID_SUPPLIER_NOTES", "supplier notes are too long")
	}
	if input.BalanceAccountIDSet && input.BalanceAccountID != nil && input.BalanceAPIKey != nil && strings.TrimSpace(*input.BalanceAPIKey) != "" {
		return infraerrors.BadRequest("SUPPLIER_BALANCE_SOURCE_CONFLICT", "choose one balance credential source")
	}
	if input.BalanceAccountIDSet {
		if input.BalanceAccountID != nil {
			if *input.BalanceAccountID <= 0 {
				return infraerrors.BadRequest("INVALID_SUPPLIER_BALANCE_ACCOUNT", "invalid balance account")
			}
			account, err := s.accounts.GetByID(ctx, *input.BalanceAccountID)
			if err != nil {
				return err
			}
			if supplier.ID == 0 || account.SupplierID == nil || *account.SupplierID != supplier.ID || !IsUpstreamBillingProbeIdentity(account.Platform, account.Type) {
				return infraerrors.BadRequest("INVALID_SUPPLIER_BALANCE_ACCOUNT", "balance account must be an API key belonging to this supplier")
			}
			supplier.BalanceAPIKey = ""
		}
		supplier.BalanceAccountID = input.BalanceAccountID
	}
	if input.BalanceAPIKey != nil {
		key := strings.TrimSpace(*input.BalanceAPIKey)
		if len(key) > MaxAPIKeyCredentialBytes || strings.ContainsAny(key, "\r\n") {
			return infraerrors.BadRequest("INVALID_SUPPLIER_BALANCE_KEY", "invalid balance API key")
		}
		supplier.BalanceAPIKey = key
		if key != "" {
			supplier.BalanceAccountID = nil
		}
	}
	supplier.HasBalanceAPIKey = supplier.BalanceAPIKey != ""
	return nil
}

func (s *SupplierService) Delete(ctx context.Context, id int64) error { return s.repo.Delete(ctx, id) }

// RefreshBalance never interprets key quota or subscription allowance as a shared wallet.
func (s *SupplierService) RefreshBalance(ctx context.Context, id int64) (*Supplier, error) {
	return s.refreshBalance(ctx, id, 0)
}

func (s *SupplierService) refreshBalance(ctx context.Context, id, preferredAccountID int64) (*Supplier, error) {
	supplier, err := s.repo.BeginBalanceRefresh(ctx, id)
	if err != nil {
		return nil, err
	}
	balance, status, code := s.fetchBalance(ctx, supplier, preferredAccountID)
	accepted, err := s.repo.SaveBalanceSnapshot(ctx, id, supplier.Revision, supplier.BalanceProbeSequence, balance, status, code, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	if !accepted {
		return nil, ErrSupplierChanged
	}
	return s.repo.GetByID(ctx, id)
}

const supplierBalanceAutoMaxAttempts = 3

func (s *SupplierService) fetchBalance(ctx context.Context, supplier *Supplier, preferredAccountID int64) (*float64, string, string) {
	if s.test == nil || s.test.httpUpstream == nil {
		return nil, "error", "transport_unavailable"
	}
	// Automatic fallbacks share the existing request budget instead of extending
	// account saves by one full timeout for each invalid credential.
	ctx, cancel := context.WithTimeout(ctx, upstreamBillingProbeRequestTimeout)
	defer cancel()
	if supplier.BalanceAccountID != nil {
		if s.accounts == nil {
			return nil, "unavailable", "balance_account_unavailable"
		}
		account, err := s.accounts.GetByID(ctx, *supplier.BalanceAccountID)
		if err != nil || !supplierBalanceAccountEligible(account, supplier.ID) {
			return nil, "unavailable", "balance_account_unavailable"
		}
		return s.fetchBalanceWithAccount(ctx, supplier, account)
	}
	if supplier.BalanceAPIKey != "" {
		return s.fetchBalanceWithAccount(ctx, supplier, nil)
	}
	if s.accounts == nil {
		return nil, "unavailable", "balance_key_missing"
	}
	accounts, err := s.accounts.ListAllWithFilters(WithSupplierAccountFilter(ctx, supplier.ID), "", AccountTypeAPIKey, "", "", 0, "")
	if err != nil {
		return nil, "error", "balance_accounts_unavailable"
	}
	// A saved key gets the first attempt. Manual refreshes start with the newest
	// account, then fall back deterministically to older supplier keys.
	sort.Slice(accounts, func(i, j int) bool {
		if (accounts[i].ID == preferredAccountID) != (accounts[j].ID == preferredAccountID) {
			return accounts[i].ID == preferredAccountID
		}
		return accounts[i].ID > accounts[j].ID
	})
	status, code, attempts := "unavailable", "balance_key_missing", 0
	for i := range accounts {
		account := &accounts[i]
		if !supplierBalanceAccountEligible(account, supplier.ID) || account.GetCredential("api_key") == "" {
			continue
		}
		if ctx.Err() != nil {
			return nil, "error", "request_failed"
		}
		if attempts >= supplierBalanceAutoMaxAttempts {
			break
		}
		attempts++
		balance, candidateStatus, candidateCode := s.fetchBalanceWithAccount(ctx, supplier, account)
		if candidateStatus == "success" {
			return balance, candidateStatus, candidateCode
		}
		if status != "error" || candidateStatus == "error" {
			status, code = candidateStatus, candidateCode
		}
	}
	return nil, status, code
}

func supplierBalanceAccountEligible(account *Account, supplierID int64) bool {
	return account != nil && account.SupplierID != nil && *account.SupplierID == supplierID && IsUpstreamBillingProbeIdentity(account.Platform, account.Type)
}

func (s *SupplierService) fetchBalanceWithAccount(ctx context.Context, supplier *Supplier, account *Account) (*float64, string, string) {
	key := supplier.BalanceAPIKey
	accountID, concurrency, proxyURL := -supplier.ID, 1, ""
	var profile *tlsfingerprint.Profile
	if account != nil {
		key, accountID, concurrency = account.GetCredential("api_key"), account.ID, account.Concurrency
		if account.ProxyID != nil {
			if account.Proxy == nil || account.Proxy.ID != *account.ProxyID {
				return nil, "error", "proxy_unavailable"
			}
			proxyURL = account.Proxy.URL()
		}
		if s.test.tlsFPProfileService != nil {
			profile = s.test.tlsFPProfileService.ResolveTLSProfile(account)
		}
	}
	if key == "" {
		return nil, "unavailable", "balance_key_missing"
	}
	base, err := s.test.validateUpstreamBaseURL(supplier.BaseURL)
	if err != nil {
		return nil, "error", "invalid_base_url"
	}
	requestCtx := WithHTTPUpstreamRedirectsDisabled(WithHTTPUpstreamProfile(ctx, HTTPUpstreamProfileDefault))
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, buildOpenAIEndpointURL(base, "/v1/usage"), nil)
	if err != nil {
		return nil, "error", "request_build_failed"
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")
	resp, err := s.test.httpUpstream.DoWithTLS(req, proxyURL, accountID, concurrency, profile)
	if err != nil {
		return nil, "error", "request_failed"
	}
	if resp == nil || resp.Body == nil {
		return nil, "error", "empty_response"
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
		return nil, "unavailable", "unsupported"
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "error", "http_error"
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, upstreamBillingProbeMaxBodyBytes+1))
	if err != nil || len(body) > upstreamBillingProbeMaxBodyBytes {
		return nil, "error", "invalid_response"
	}
	return parseSupplierWalletBalance(body)
}

func parseSupplierWalletBalance(body []byte) (*float64, string, string) {
	var response struct {
		Mode         string          `json:"mode"`
		Unit         string          `json:"unit"`
		Balance      *float64        `json:"balance"`
		Subscription json.RawMessage `json:"subscription"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, "error", "invalid_response"
	}
	if response.Mode != "unrestricted" || response.Balance == nil || (len(response.Subscription) != 0 && string(response.Subscription) != "null") {
		return nil, "unavailable", "wallet_not_provided"
	}
	if response.Unit != "USD" || math.IsNaN(*response.Balance) || math.IsInf(*response.Balance, 0) {
		return nil, "error", "invalid_response"
	}
	return response.Balance, "success", ""
}

// RefreshAfterAccountSave observes current upstream data without enabling periodic probes.
// Probe failures do not roll back a successfully saved account.
func (s *SupplierService) RefreshAfterAccountSave(ctx context.Context, account *Account) {
	if account == nil || account.SupplierID == nil {
		return
	}
	accountID, supplierID := account.ID, *account.SupplierID
	AfterAccountCommit(ctx, func(postCommitCtx context.Context) {
		var probes sync.WaitGroup
		if s.billing != nil {
			probes.Add(1)
			go func() { defer probes.Done(); _, _ = s.billing.ProbeAccount(postCommitCtx, accountID) }()
		}
		_, _ = s.refreshBalance(postCommitCtx, supplierID, accountID)
		probes.Wait()
	})
}
