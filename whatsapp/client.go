package whatsapp

import (
	"context"
	"fmt"
	"sync"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"go.uber.org/zap"

	"whatsapp-gateway/service"
)

type Client struct {
	wac       *whatsmeow.Client
	alert     *service.AlertService
	logger    *zap.Logger
	mu        sync.Mutex
	currentQR string
}

func New(ctx context.Context, store *sqlstore.Container, alert *service.AlertService, logger *zap.Logger) (*Client, error) {
	deviceStore, err := store.GetFirstDevice(ctx)
	if err != nil {
		return nil, err
	}

	wac := whatsmeow.NewClient(deviceStore, waLog.Noop)
	c := &Client{
		wac:    wac,
		alert:  alert,
		logger: logger,
	}
	wac.AddEventHandler(c.handleEvent)
	return c, nil
}

func (c *Client) Connect(ctx context.Context) error {
	if c.wac.Store.ID == nil {
		qrChan, err := c.wac.GetQRChannel(ctx)
		if err != nil {
			return err
		}
		if err := c.wac.Connect(); err != nil {
			return err
		}
		go c.consumeQR(qrChan)
	} else {
		if err := c.wac.Connect(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) consumeQR(ch <-chan whatsmeow.QRChannelItem) {
	for evt := range ch {
		if evt.Event == "code" {
			c.mu.Lock()
			c.currentQR = evt.Code
			c.mu.Unlock()
		}
	}
}

func (c *Client) GetQRCode() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.currentQR == "" {
		return "", fmt.Errorf("QR not available â€” not logged in or already connected")
	}
	return c.currentQR, nil
}

func (c *Client) IsConnected() bool {
	return c.wac.IsConnected() && c.wac.Store.ID != nil
}

func (c *Client) GetPhone() string {
	if c.wac.Store.ID == nil {
		return ""
	}
	return c.wac.Store.ID.User
}

func (c *Client) WAClient() *whatsmeow.Client {
	return c.wac
}

func (c *Client) Disconnect() {
	c.wac.Disconnect()
}

func (c *Client) Logout(ctx context.Context) error {
	err := c.wac.Logout(ctx)
	c.mu.Lock()
	c.currentQR = ""
	c.mu.Unlock()
	return err
}

func (c *Client) handleEvent(evt interface{}) {
	switch evt.(type) {
	case *events.Connected:
		c.logger.Info("whatsapp connected")
		c.mu.Lock()
		c.currentQR = ""
		c.mu.Unlock()
	case *events.Disconnected:
		c.logger.Warn("whatsapp disconnected")
		c.alert.Notify("whatsapp connection lost")
	}
}

