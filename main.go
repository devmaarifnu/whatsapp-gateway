package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/lumberjack.v2"

	"whatsapp-gateway/config"
	"whatsapp-gateway/db"
	"whatsapp-gateway/handler"
	"whatsapp-gateway/middleware"
	"whatsapp-gateway/repository"
	"whatsapp-gateway/service"
	"whatsapp-gateway/whatsapp"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		// belum ada logger — fallback ke panic
		panic("load config failed: " + err.Error())
	}

	logger := newLogger(cfg.Log)
	defer logger.Sync()

	mysqlDB, err := db.NewMySQL(cfg.Database.MySQL)
	if err != nil {
		logger.Fatal("mysql connect failed", zap.Error(err))
	}
	defer mysqlDB.Close()

	if err := db.MigrateMySQL(mysqlDB); err != nil {
		logger.Fatal("mysql migrate failed", zap.Error(err))
	}

	ctx := context.Background()
	sqliteStore, err := db.NewSQLiteStore(ctx, cfg.Database.SQLite)
	if err != nil {
		logger.Fatal("sqlite open failed", zap.Error(err))
	}

	tokenRepo := repository.NewTokenRepo(mysqlDB)
	templateRepo := repository.NewTemplateRepo(mysqlDB)
	messageRepo := repository.NewMessageRepo(mysqlDB)

	alertSvc := service.NewAlertService(cfg.Alert, logger)
	templateSvc := service.NewTemplateService(templateRepo)

	waClient, err := whatsapp.New(ctx, sqliteStore, alertSvc, logger)
	if err != nil {
		logger.Fatal("whatsapp init failed", zap.Error(err))
	}
	if err := waClient.Connect(ctx); err != nil {
		logger.Fatal("whatsapp connect failed", zap.Error(err))
	}

	msgSvc := service.NewMessageService(messageRepo, templateSvc, waClient.WAClient(), cfg.Worker.Count, logger)

	// Handler pesan masuk — aktifkan bot perintah & simpan inbound ke MySQL.
	// Didaftarkan lewat callback agar whatsapp.Client tidak perlu import handler.
	incomingRepo := repository.NewIncomingRepo(mysqlDB)
	msgHandler := handler.NewMessageHandler(logger, incomingRepo)
	waClient.SetOnMessage(msgHandler.Handle)

	sendH := handler.NewSendHandler(msgSvc)
	historyH := handler.NewHistoryHandler(messageRepo)
	waH := handler.NewWAHandler(waClient)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	}))

	bp := cfg.Server.BasePath
	r.StaticFile(bp+"/qr.html", "./qr.html")

	r.NoRoute(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if c.Request.Method == http.MethodOptions {
			c.Status(http.StatusNoContent)
			return
		}
		c.Status(http.StatusNotFound)
	})

	auth := middleware.Auth(tokenRepo)
	api := r.Group(bp+"/", auth)
	{
		api.POST("send/message", sendH.SendMessage)
		api.POST("send/template", sendH.SendTemplate)
		api.GET("messages/:id", historyH.GetMessage)
		api.GET("messages", historyH.ListMessages)
		api.GET("qr", waH.GetQR)
		api.GET("status", waH.GetStatus)
		api.POST("logout", waH.Logout)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: r,
	}

	go func() {
		logger.Info("server started", zap.Int("port", cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("server error", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
	waClient.Disconnect()
}

func newLogger(cfg config.LogConfig) *zap.Logger {
	encCfg := zap.NewProductionEncoderConfig()
	encCfg.EncodeTime = zapcore.ISO8601TimeEncoder

	level := zapcore.InfoLevel
	_ = level.UnmarshalText([]byte(cfg.Level))

	jsonEnc := zapcore.NewJSONEncoder(encCfg)

	// stdout core
	cores := []zapcore.Core{
		zapcore.NewCore(jsonEnc, zapcore.AddSync(os.Stdout), level),
	}

	// file core (jika log.file dikonfigurasi)
	if cfg.File != "" {
		rotator := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAgeDays,
			Compress:   cfg.Compress,
		}
		cores = append(cores, zapcore.NewCore(jsonEnc, zapcore.AddSync(rotator), level))
	}

	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
