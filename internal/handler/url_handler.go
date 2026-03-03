package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"

	"tinLink/internal/service"
)

var tracer = otel.Tracer("handler")

type URLHandler struct {
	urlService *service.URLService
}

func NewURLHandler(urlService *service.URLService) *URLHandler {
	return &URLHandler{urlService: urlService}
}

type ShortenRequest struct {
	LongURL    string `json:"long_url" binding:"required,url,max=2048"`
	CustomCode string `json:"custom_code,omitempty" binding:"omitempty,alphanum,min=4,max=10"`
	ExpireDays int    `json:"expire_days,omitempty" binding:"omitempty,min=1,max=3650"`
}

type ShortenResponse struct {
	ShortCode string `json:"short_code"`
	ShortURL  string `json:"short_url"`
	LongURL   string `json:"long_url"`
	ExpireAt  string `json:"expire_at,omitempty"`
}

// Shorten 创建短链接
// @Summary 创建短链接
// @Tags URL
// @Accept json
// @Produce json
// @Param request body ShortenRequest true "请求参数"
// @Success 201 {object} Response{data=ShortenResponse}
// @Router /api/v1/shorten [post]
func (h *URLHandler) Shorten(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "URLHandler.Shorten")
	defer span.End()

	var req ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		span.RecordError(err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "Invalid request: " + err.Error(),
		})
		return
	}

	// 规范化URL
	req.LongURL = strings.TrimSpace(req.LongURL)
	if req.ExpireDays <= 0 {
		req.ExpireDays = 365
	}

	span.SetAttributes(
		attribute.String("long_url", req.LongURL),
		attribute.Int("expire_days", req.ExpireDays),
	)

	url, err := h.urlService.CreateShortURL(ctx, service.CreateURLRequest{
		LongURL:    req.LongURL,
		CustomCode: req.CustomCode,
		ExpireDays: req.ExpireDays,
	})
	if err != nil {
		span.RecordError(err)
		switch err {
		case service.ErrURLExists:
			c.JSON(http.StatusConflict, Response{Code: 409, Message: "Short code already exists"})
		case service.ErrInvalidCode:
			c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "Invalid custom code"})
		case service.ErrCircuitOpen:
			c.JSON(http.StatusServiceUnavailable, Response{Code: 503, Message: "Service temporarily unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "Internal server error"})
		}
		return
	}

	// 构建完整URL
	scheme := "http"
	if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	shortURL := scheme + "://" + c.Request.Host + "/" + url.ShortCode

	c.JSON(http.StatusCreated, Response{
		Code:    0,
		Message: "success",
		Data: ShortenResponse{
			ShortCode: url.ShortCode,
			ShortURL:  shortURL,
			LongURL:   url.LongURL,
			ExpireAt:  url.ExpireAt.Format("2006-01-02 15:04:05"),
		},
	})
}

// Redirect 短链接跳转 - 核心高频接口
func (h *URLHandler) Redirect(c *gin.Context) {
	ctx, span := tracer.Start(c.Request.Context(), "URLHandler.Redirect")
	defer span.End()

	code := c.Param("code")
	if code == "" || len(code) > 10 {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "Invalid code"})
		return
	}

	span.SetAttributes(attribute.String("short_code", code))

	longURL, err := h.urlService.GetLongURL(ctx, code)
	if err != nil {
		span.RecordError(err)
		switch err {
		case service.ErrURLNotFound:
			c.JSON(http.StatusNotFound, Response{Code: 404, Message: "Short URL not found"})
		case service.ErrURLExpired:
			c.JSON(http.StatusGone, Response{Code: 410, Message: "Short URL has expired"})
		case service.ErrCircuitOpen:
			c.JSON(http.StatusServiceUnavailable, Response{Code: 503, Message: "Service temporarily unavailable"})
		default:
			c.JSON(http.StatusInternalServerError, Response{Code: 500, Message: "Internal server error"})
		}
		return
	}

	// 异步记录访问（不阻塞跳转）
	go h.urlService.RecordAccess(ctx, code, c.ClientIP(), c.Request.UserAgent(), c.Request.Referer())

	// 302临时重定向（便于统计）
	c.Redirect(http.StatusFound, longURL)
}

// GetURL 获取短链接详情
func (h *URLHandler) GetURL(c *gin.Context) {
	code := c.Param("code")
	url, err := h.urlService.GetURLDetail(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Message: "Not found"})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Data: url})
}

// DeleteURL 删除短链接
func (h *URLHandler) DeleteURL(c *gin.Context) {
	code := c.Param("code")
	if err := h.urlService.DeleteURL(c.Request.Context(), code); err != nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Message: "Not found"})
		return
	}
	c.JSON(http.StatusOK, Response{Code: 0, Message: "Deleted"})
}

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}
