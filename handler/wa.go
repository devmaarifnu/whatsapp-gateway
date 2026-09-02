package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	wa "whatsapp-gateway/whatsapp"
)

type WAHandler struct {
	client *wa.Client
}

func NewWAHandler(client *wa.Client) *WAHandler {
	return &WAHandler{client: client}
}

func (h *WAHandler) GetQR(c *gin.Context) {
	if h.client.IsConnected() {
		c.JSON(http.StatusOK, gin.H{"connected": true})
		return
	}

	code, err := h.client.GetQRCode()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"connected": false,
			"error":     err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"connected": false,
		"code":      code,
	})
}

func (h *WAHandler) Logout(c *gin.Context) {
	if err := h.client.Logout(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "logged out — restart server to scan new QR"})
}

func (h *WAHandler) GetStatus(c *gin.Context) {
	connected := h.client.IsConnected()
	phone := ""
	if connected {
		phone = h.client.GetPhone()
	}
	c.JSON(http.StatusOK, gin.H{
		"connected": connected,
		"phone":     phone,
	})
}
