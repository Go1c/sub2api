package admin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
)

const maxInvoiceUploadBytes = 10 << 20

type InvoiceHandler struct {
	invoiceService *service.InvoiceService
}

func NewInvoiceHandler(invoiceService *service.InvoiceService) *InvoiceHandler {
	return &InvoiceHandler{invoiceService: invoiceService}
}

func (h *InvoiceHandler) List(c *gin.Context) {
	page, pageSize := response.ParsePagination(c)
	status := strings.TrimSpace(c.Query("status"))
	search := strings.TrimSpace(c.Query("search"))
	var userID int64
	if raw := strings.TrimSpace(c.Query("user_id")); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed <= 0 {
			response.BadRequest(c, "Invalid user_id")
			return
		}
		userID = parsed
	}
	items, result, err := h.invoiceService.ListAdmin(c.Request.Context(), pagination.PaginationParams{Page: page, PageSize: pageSize}, service.InvoiceListFilter{
		Search: search,
		Status: status,
		UserID: userID,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Paginated(c, dto.InvoiceRequestsFromService(items), result.Total, result.Page, result.PageSize)
}

func (h *InvoiceHandler) Complete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	existing, err := h.invoiceService.Get(c.Request.Context(), id)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	if existing.Status != service.InvoiceStatusProcessing {
		response.ErrorFrom(c, service.ErrInvoiceInvalidStatus)
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Invoice file is required")
		return
	}
	if fileHeader.Size <= 0 || fileHeader.Size > maxInvoiceUploadBytes {
		response.BadRequest(c, "Invoice file must be between 1 byte and 10MB")
		return
	}
	fileName := sanitizeInvoiceFileName(fileHeader.Filename)
	if !allowedInvoiceFileName(fileName) {
		response.BadRequest(c, "Invoice file must be pdf, png, jpg, or jpeg")
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FILE_OPEN_FAILED: "+err.Error())
		return
	}
	defer func() { _ = file.Close() }()
	bytes, err := io.ReadAll(io.LimitReader(file, maxInvoiceUploadBytes+1))
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FILE_READ_FAILED: "+err.Error())
		return
	}
	if len(bytes) == 0 || len(bytes) > maxInvoiceUploadBytes {
		response.BadRequest(c, "Invoice file must be between 1 byte and 10MB")
		return
	}

	taxRate := 0.01
	if raw := strings.TrimSpace(c.PostForm("tax_rate")); raw != "" {
		parsed, err := strconv.ParseFloat(raw, 64)
		if err != nil || parsed < 0 {
			response.BadRequest(c, "Invalid tax_rate")
			return
		}
		taxRate = parsed
	}

	storedPath, err := storeInvoiceUpload(existing.OrderNo, fileName, bytes)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "FILE_SAVE_FAILED: "+err.Error())
		return
	}
	contentType := fileHeader.Header.Get("Content-Type")
	if contentType == "" {
		contentType = http.DetectContentType(bytes)
	}

	item, err := h.invoiceService.Complete(c.Request.Context(), id, service.CompleteInvoiceInput{
		FileName:    fileName,
		FilePath:    storedPath,
		FileSize:    int64(len(bytes)),
		ContentType: contentType,
		Bytes:       bytes,
		TaxRate:     &taxRate,
	})
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.InvoiceRequestFromService(item))
}

func (h *InvoiceHandler) Fail(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "Invalid invoice ID")
		return
	}
	var req dto.FailInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	item, err := h.invoiceService.MarkFailed(c.Request.Context(), id, req.Reason)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	response.Success(c, dto.InvoiceRequestFromService(item))
}

func (h *InvoiceHandler) Download(c *gin.Context) {
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
	serveAdminInvoiceFile(c, item)
}

func serveAdminInvoiceFile(c *gin.Context, item *service.InvoiceRequest) {
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

func storeInvoiceUpload(orderNo, fileName string, bytes []byte) (string, error) {
	dir := filepath.Join("data", "invoices", orderNo)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	storedName := fmt.Sprintf("%d-%s", time.Now().UnixNano(), fileName)
	path := filepath.Join(dir, storedName)
	if err := os.WriteFile(path, bytes, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

func sanitizeInvoiceFileName(name string) string {
	name = filepath.Base(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, "\x00", "")
	if name == "." || name == string(filepath.Separator) || name == "" {
		return "invoice.pdf"
	}
	return name
}

func allowedInvoiceFileName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".pdf", ".png", ".jpg", ".jpeg":
		return true
	default:
		return false
	}
}
