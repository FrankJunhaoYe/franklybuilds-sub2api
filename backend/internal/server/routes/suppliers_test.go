package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSupplierAccountFilterRejectsMalformedScope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, query := range []string{"", "-1", "invalid", "9223372036854775808"} {
		router := gin.New()
		called := false
		router.GET("/accounts", supplierAccountFilterMiddleware, func(c *gin.Context) { called = true; c.Status(http.StatusOK) })
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/accounts?supplier_id="+query, nil))
		require.Equal(t, http.StatusBadRequest, recorder.Code)
		require.False(t, called)
	}
}

func TestSupplierAccountFilterPreservesSelfManagedAndSupplierScopes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, test := range []struct {
		path    string
		id      int64
		present bool
	}{{"/accounts", 0, false}, {"/accounts?supplier_id=0", 0, true}, {"/accounts?supplier_id=7", 7, true}} {
		router := gin.New()
		router.GET("/accounts", supplierAccountFilterMiddleware, func(c *gin.Context) {
			id, present := service.SupplierAccountFilterFromContext(c.Request.Context())
			require.Equal(t, test.id, id)
			require.Equal(t, test.present, present)
			c.Status(http.StatusOK)
		})
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, test.path, nil))
		require.Equal(t, http.StatusOK, recorder.Code)
	}
}
