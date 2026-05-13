package handler

import (
	"net/http"
	"os"
	"strconv"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

func (h *InvoiceHandler) Overview(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	overview, err := h.invoiceService.Overview(c.Request.Context(), subject.UserID)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, overview)
}

func (h *InvoiceHandler) List(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	page, pageSize := response.ParsePagination(c)
	items, result, err := h.invoiceService.ListByUser(c.Request.Context(), subject.UserID, pagination.PaginationParams{Page: page, PageSize: pageSize})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.InvoiceRequestsFromService(items), result.Total, result.Page, result.PageSize)
}

func (h *InvoiceHandler) Create(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	var req dto.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.invoiceService.Create(c.Request.Context(), subject.UserID, service.CreateInvoiceRequestInput{
		Title:          req.Title,
		TaxNumber:      req.TaxNumber,
		Amount:         req.Amount,
		RecipientEmail: req.RecipientEmail,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.InvoiceRequestFromService(item))
}

func (h *InvoiceHandler) Download(c *gin.Context) {
	subject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "Unauthorized")
		return
	}
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	item, err := h.invoiceService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if item.UserID != subject.UserID {
		response.Forbidden(c, "Forbidden")
		return
	}
	serveInvoiceFile(c, item)
}

func serveInvoiceFile(c *gin.Context, item *service.InvoiceRequest) {
	if item == nil || item.FilePath == "" {
		response.NotFound(c, "Invoice file not found")
		return
	}
	if _, err := os.Stat(item.FilePath); err != nil {
		if os.IsNotExist(err) {
			response.NotFound(c, "Invoice file not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "FILE_ERROR: "+err.Error())
		return
	}
	c.FileAttachment(item.FilePath, item.FileName)
}
