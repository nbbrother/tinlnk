package handler

import (
	"net/http"
	"strconv"

	"tinLink/internal/service"

	"github.com/gin-gonic/gin"
)

type StatsHandler struct {
	statsService *service.StatsService
}

func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// GetStats 获取短链接统计
func (h *StatsHandler) GetStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, Response{Code: 400, Message: "Code required"})
		return
	}

	stats, err := h.statsService.GetStats(c.Request.Context(), code)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Message: "Not found"})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code: 0,
		Data: gin.H{
			"short_code":  code,
			"total_pv":    stats.TotalPV,
			"total_uv":    stats.TotalUV,
			"today_pv":    stats.TodayPV,
			"today_uv":    stats.TodayUV,
			"created_at":  stats.CreatedAt.Format("2006-01-02 15:04:05"),
			"last_access": stats.LastAccess.Format("2006-01-02 15:04:05"),
		},
	})
}

// GetDailyStats 获取每日统计
func (h *StatsHandler) GetDailyStats(c *gin.Context) {
	code := c.Param("code")
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	if days <= 0 || days > 90 {
		days = 7
	}

	stats, err := h.statsService.GetDailyStats(c.Request.Context(), code, days)
	if err != nil {
		c.JSON(http.StatusNotFound, Response{Code: 404, Message: "Not found"})
		return
	}

	c.JSON(http.StatusOK, Response{Code: 0, Data: stats})
}
