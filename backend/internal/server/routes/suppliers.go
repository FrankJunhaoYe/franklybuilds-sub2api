package routes

import (
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func supplierAccountFilterMiddleware(c *gin.Context) {
	if value, provided := c.GetQuery("supplier_id"); provided {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id < 0 {
			response.BadRequest(c, "Invalid supplier ID")
			c.Abort()
			return
		}
		c.Request = c.Request.WithContext(service.WithSupplierAccountFilter(c.Request.Context(), id))
	}
	c.Next()
}

func registerSupplierRoutes(admin *gin.RouterGroup, h *handler.Handlers) {
	if h.Admin.Supplier == nil {
		return
	}
	suppliers := admin.Group("/suppliers")
	suppliers.GET("", h.Admin.Supplier.List)
	suppliers.POST("", h.Admin.Supplier.Create)
	suppliers.GET("/:id", h.Admin.Supplier.Get)
	suppliers.PUT("/:id", h.Admin.Supplier.Update)
	suppliers.DELETE("/:id", h.Admin.Supplier.Delete)
	suppliers.POST("/:id/refresh-balance", h.Admin.Supplier.RefreshBalance)
}
