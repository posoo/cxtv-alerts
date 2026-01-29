package handler

import (
	"net/http"
	"sort"
	"strconv"

	"cxtv-alerts/internal/service"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	svc *service.Service
}

func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	api := r.Group("/api")
	{
		api.GET("/streamers", h.GetStreamers)
		api.GET("/history/:id", h.GetHistory)
		api.GET("/stats/:id", h.GetStats)
	}
}

func (h *Handler) GetStreamers(c *gin.Context) {
	streamers := h.svc.GetStreamers()

	// Sort: live streamers first, then by name
	sort.Slice(streamers, func(i, j int) bool {
		if streamers[i].IsLive != streamers[j].IsLive {
			return streamers[i].IsLive
		}
		return streamers[i].Name < streamers[j].Name
	})

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": streamers,
	})
}

func (h *Handler) GetHistory(c *gin.Context) {
	id := c.Param("id")
	limit := 50

	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	history, err := h.svc.GetHistory(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": history,
	})
}

func (h *Handler) GetStats(c *gin.Context) {
	id := c.Param("id")

	stats, err := h.svc.GetStats(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    1,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": stats,
	})
}
