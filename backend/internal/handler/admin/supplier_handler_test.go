package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupplierWriteRejectsUnknownFieldsWithoutEchoingSecrets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{`{"name":"source","balance":999}`, `{"name":"source","unknown":"test-secret"}`, `{"name":"source"} {"name":"second"}`} {
		recorder := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(recorder)
		c.Request = httptest.NewRequest(http.MethodPost, "/api/v1/admin/suppliers", strings.NewReader(body))
		_, ok := decodeSupplierWrite(c)
		require.False(t, ok)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.NotContains(t, recorder.Body.String(), "test-secret")
	}
}

func TestSupplierWriteDistinguishesOmittedAndClearedCredentials(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, body := range []string{`{}`, `{"balance_account_id":null,"balance_api_key":""}`} {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest(http.MethodPut, "/api/v1/admin/suppliers/1", strings.NewReader(body))
		input, ok := decodeSupplierWrite(c)
		require.True(t, ok)
		if body == `{}` {
			require.False(t, input.BalanceAccountIDSet)
			require.Nil(t, input.BalanceAPIKey)
		} else {
			require.True(t, input.BalanceAccountIDSet)
			require.Nil(t, input.BalanceAccountID)
			require.NotNil(t, input.BalanceAPIKey)
			require.Empty(t, *input.BalanceAPIKey)
		}
	}
}

func TestSupplierRejectsScopedAdminBeforeAccessingRepository(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(http.MethodGet, "/api/v1/admin/suppliers", nil).WithContext(adminaccess.WithPrincipal(context.Background(), &adminaccess.Principal{KeyID: "restricted"}))
	NewSupplierHandler(nil).List(c)
	require.Equal(t, http.StatusForbidden, recorder.Code)
}

type supplierRefreshCounter struct {
	calls   int
	account *service.Account
}

func (r *supplierRefreshCounter) RefreshAfterAccountSave(context.Context, *service.Account) {
	r.calls++
	rate := 0.6
	r.account.RateMultiplier = &rate
}

func TestSupplierAccountSaveRefreshesOnceAndReturnsLatestAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, method := range []string{http.MethodPost, http.MethodPut} {
		for _, owned := range []bool{false, true} {
			name := method + "/own"
			if owned {
				name = method + "/supplier"
			}
			t.Run(name, func(t *testing.T) {
				rate := 1.0
				account := service.Account{ID: 3, Name: "key", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive, RateMultiplier: &rate}
				if owned {
					supplierID := int64(7)
					account.SupplierID = &supplierID
				}
				svc := &simpleModeAccountService{stubAdminService: newStubAdminService(), account: account}
				h := NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
				refresher := &supplierRefreshCounter{account: &svc.account}
				h.suppliers = refresher
				router := gin.New()
				router.POST("/accounts", h.Create)
				router.PUT("/accounts/:id", h.Update)
				path := "/accounts"
				if method == http.MethodPut {
					path += "/3"
				}
				body := `{"name":"key","platform":"anthropic","type":"apikey","credentials":{"api_key":"test-only"}}`
				request := httptest.NewRequest(method, path, strings.NewReader(body))
				request.Header.Set("Content-Type", "application/json")
				result := httptest.NewRecorder()
				router.ServeHTTP(result, request)
				require.Equal(t, http.StatusOK, result.Code, result.Body.String())
				var payload struct {
					Data struct {
						RateMultiplier float64 `json:"rate_multiplier"`
					} `json:"data"`
				}
				require.NoError(t, json.Unmarshal(result.Body.Bytes(), &payload))
				if owned {
					require.Equal(t, 1, refresher.calls)
					require.InDelta(t, 0.6, payload.Data.RateMultiplier, 0.0001)
				} else {
					require.Zero(t, refresher.calls)
					require.Equal(t, 1.0, payload.Data.RateMultiplier)
				}
			})
		}
	}
}

type supplierCreateCaptureService struct {
	*simpleModeAccountService
	input *service.CreateAccountInput
}

func (s *supplierCreateCaptureService) CreateAccount(_ context.Context, input *service.CreateAccountInput) (*service.Account, error) {
	s.input = input
	return &s.account, nil
}

func TestSupplierAccountCreatePreservesOmittedAndExplicitDefaultFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, explicit := range []bool{false, true} {
		svc := &supplierCreateCaptureService{simpleModeAccountService: &simpleModeAccountService{stubAdminService: newStubAdminService(), account: service.Account{ID: 3, Name: "key", Platform: service.PlatformAnthropic, Type: service.AccountTypeAPIKey, Status: service.StatusActive}}}
		h := NewAccountHandler(svc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
		body := `{"supplier_id":7,"name":"key","platform":"anthropic","type":"apikey","credentials":{"api_key":"test-only"}`
		if explicit {
			body += `,"concurrency":0,"priority":0,"upstream_billing_rate_sync_enabled":false`
		}
		body += `}`
		router := gin.New()
		router.POST("/accounts", h.Create)
		request := httptest.NewRequest(http.MethodPost, "/accounts", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		result := httptest.NewRecorder()
		router.ServeHTTP(result, request)
		require.Equal(t, http.StatusOK, result.Code, result.Body.String())
		require.NotNil(t, svc.input)
		require.Equal(t, explicit, svc.input.ConcurrencySet)
		require.Equal(t, explicit, svc.input.PrioritySet)
		require.Zero(t, svc.input.Concurrency)
		require.Zero(t, svc.input.Priority)
		if explicit {
			require.NotNil(t, svc.input.RateSyncEnabled)
			require.False(t, *svc.input.RateSyncEnabled)
		} else {
			require.Nil(t, svc.input.RateSyncEnabled)
		}
	}
}
