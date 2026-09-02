package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"whatsapp-gateway/model"
	"whatsapp-gateway/repository"
)

type HistoryHandler struct {
	msgRepo *repository.MessageRepo
}

func NewHistoryHandler(msgRepo *repository.MessageRepo) *HistoryHandler {
	return &HistoryHandler{msgRepo: msgRepo}
}

func (h *HistoryHandler) GetMessage(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	msg, err := h.msgRepo.FindByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if msg == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, msg)
}

func (h *HistoryHandler) ListMessages(c *gin.Context) {
	to := c.Query("to")
	statusStr := c.Query("status")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if limit <= 0 || limit > 100 {
		limit = 50
	}

	msgs, err := h.msgRepo.List(to, model.MessageStatus(statusStr), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": msgs, "count": len(msgs)})
}

