package admin

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/adminaccess"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type SupplierHandler struct{ suppliers *service.SupplierService }

func NewSupplierHandler(suppliers *service.SupplierService) *SupplierHandler {
	return &SupplierHandler{suppliers: suppliers}
}

type supplierWriteRequest struct {
	Name             *string                `json:"name"`
	BaseURL          *string                `json:"base_url"`
	Status           *string                `json:"status"`
	Notes            *string                `json:"notes"`
	BalanceAccountID dto.NullableInt64Field `json:"balance_account_id"`
	BalanceAPIKey    *string                `json:"balance_api_key"`
}

func requireFullSupplierAdmin(c *gin.Context) bool {
	if adminaccess.FromContext(c.Request.Context()) != nil {
		response.Forbidden(c, "Supplier management requires a full administrator")
		return false
	}
	return true
}

func supplierRequestID(c *gin.Context) (int64, bool) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		response.BadRequest(c, "Invalid supplier ID")
		return 0, false
	}
	return id, true
}

func decodeSupplierWrite(c *gin.Context) (service.SupplierInput, bool) {
	var req supplierWriteRequest
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		response.BadRequest(c, "Invalid supplier request")
		return service.SupplierInput{}, false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		response.BadRequest(c, "Invalid supplier request")
		return service.SupplierInput{}, false
	}
	return service.SupplierInput{Name: req.Name, BaseURL: req.BaseURL, Status: req.Status, Notes: req.Notes, BalanceAccountID: req.BalanceAccountID.Value, BalanceAccountIDSet: req.BalanceAccountID.Set, BalanceAPIKey: req.BalanceAPIKey}, true
}

func (h *SupplierHandler) List(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	page, size := response.ParsePagination(c)
	items, result, err := h.suppliers.List(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: size}, c.Query("status"), c.Query("search"))
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, items, result.Total, page, size)
}

func (h *SupplierHandler) Get(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	id, ok := supplierRequestID(c)
	if !ok {
		return
	}
	item, err := h.suppliers.GetByID(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierHandler) Create(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	input, ok := decodeSupplierWrite(c)
	if !ok {
		return
	}
	item, err := h.suppliers.Create(c.Request.Context(), input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierHandler) Update(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	id, ok := supplierRequestID(c)
	if !ok {
		return
	}
	input, ok := decodeSupplierWrite(c)
	if !ok {
		return
	}
	item, err := h.suppliers.Update(c.Request.Context(), id, input)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}

func (h *SupplierHandler) Delete(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	id, ok := supplierRequestID(c)
	if !ok {
		return
	}
	if err := h.suppliers.Delete(c.Request.Context(), id); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, gin.H{"message": "Supplier deleted successfully"})
}

func (h *SupplierHandler) RefreshBalance(c *gin.Context) {
	if !requireFullSupplierAdmin(c) {
		return
	}
	id, ok := supplierRequestID(c)
	if !ok {
		return
	}
	item, err := h.suppliers.RefreshBalance(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, item)
}
