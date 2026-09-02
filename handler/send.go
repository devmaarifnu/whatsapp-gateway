package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"whatsapp-gateway/service"
)

type SendHandler struct {
	msgSvc *service.MessageService
}

func NewSendHandler(msgSvc *service.MessageService) *SendHandler {
	return &SendHandler{msgSvc: msgSvc}
}

type sendMessageReq struct {
	To      string `json:"to" binding:"required"`
	Message string `json:"message" binding:"required"`
}

func (h *SendHandler) SendMessage(c *gin.Context) {
	var req sendMessageReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.msgSvc.SendSingle(req.To, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message_id": id, "status": "pending"})
}

type sendTemplateReq struct {
	To        string            `json:"to" binding:"required"`
	Template  string            `json:"template" binding:"required"`
	Variables map[string]string `json:"variables"`
}

func (h *SendHandler) SendTemplate(c *gin.Context) {
	var req sendTemplateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	id, err := h.msgSvc.SendTemplate(req.To, req.Template, req.Variables)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{"message_id": id, "status": "pending"})
}

