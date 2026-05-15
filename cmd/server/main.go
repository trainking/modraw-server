package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/trainking/modraw-server/internal/config"
	"github.com/trainking/modraw-server/internal/database"
	"github.com/trainking/modraw-server/internal/handler"
	"github.com/trainking/modraw-server/internal/middleware"
	"github.com/trainking/modraw-server/internal/repository"
	"github.com/trainking/modraw-server/internal/service"
	"github.com/trainking/modraw-server/internal/ws"
	jwtpkg "github.com/trainking/modraw-server/pkg/jwt"
)

func main() {
	cfg := config.Load()

	gin.SetMode(cfg.GinMode)

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db, "migrations"); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	// WebSocket hub
	wsHub := ws.NewHub()
	go wsHub.Run()

	r := setupRouter(cfg, db, wsHub)

	srv := &http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("[server] starting on %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[server] error: %v", err)
		}
	}()

	<-quit
	log.Println("[server] shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("[server] forced shutdown: %v", err)
	}

	log.Println("[server] stopped")
}

func setupRouter(cfg *config.Config, db *sql.DB, wsHub *ws.Hub) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), middleware.Recovery(), middleware.CORS(cfg))

	// Health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Repositories
	userRepo := repository.NewUserRepository(db)
	folderRepo := repository.NewFolderRepository(db)
	canvasRepo := repository.NewCanvasRepository(db)

	wsHub.SetStateLoader(func(canvasID string) (json.RawMessage, error) {
		canvas, err := canvasRepo.GetByID(context.Background(), canvasID)
		if err != nil {
			return nil, err
		}
		return canvas.Data, nil
	})

	collaboratorRepo := repository.NewCollaboratorRepository(db)
	libraryRepo := repository.NewLibraryRepository(db)
	shareLinkRepo := repository.NewShareLinkRepository(db)

	// Services
	authService := service.NewAuthService(userRepo, cfg)
	folderService := service.NewFolderService(folderRepo)
	canvasService := service.NewCanvasService(canvasRepo)
	collaboratorService := service.NewCollaboratorService(collaboratorRepo, canvasRepo, userRepo)
	libraryService := service.NewLibraryService(libraryRepo)
	shareLinkService := service.NewShareLinkService(shareLinkRepo, canvasRepo, cfg)

	// Handlers
	authHandler := handler.NewAuthHandler(authService)
	userHandler := handler.NewUserHandler(userRepo)
	folderHandler := handler.NewFolderHandler(folderService)
	canvasHandler := handler.NewCanvasHandler(canvasService)
	collaboratorHandler := handler.NewCollaboratorHandler(collaboratorService)
	libraryHandler := handler.NewLibraryHandler(libraryService)
	shareLinkHandler := handler.NewShareLinkHandler(shareLinkService)

	// API v1
	api := r.Group("/api/v1")

	// Auth routes
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.Refresh)
	auth.DELETE("/logout", middleware.AuthRequired(cfg), authHandler.Logout)

	// User routes
	users := api.Group("/users")
	users.Use(middleware.AuthRequired(cfg))
	users.GET("/me", userHandler.Me)
	users.PUT("/me", userHandler.UpdateMe)
	users.PUT("/me/password", userHandler.ChangePassword)

	// Folder routes
	folders := api.Group("/folders")
	folders.Use(middleware.AuthRequired(cfg))
	folders.GET("", folderHandler.List)
	folders.POST("", folderHandler.Create)
	folders.GET("/:id", folderHandler.Get)
	folders.PUT("/:id", folderHandler.Update)
	folders.DELETE("/:id", folderHandler.Delete)
	folders.PUT("/:id/move", folderHandler.Move)

	// Canvas routes
	canvases := api.Group("/canvases")
	canvases.Use(middleware.AuthRequired(cfg))
	canvases.GET("", canvasHandler.List)
	canvases.POST("", canvasHandler.Create)
	canvases.GET("/:id", canvasHandler.Get)
	canvases.PUT("/:id", canvasHandler.Update)
	canvases.DELETE("/:id", canvasHandler.Delete)
	canvases.PUT("/:id/data", canvasHandler.SaveData)
	canvases.PUT("/:id/move", canvasHandler.Move)

	// Collaborator routes
	collaborators := canvases.Group("/:id/collaborators")
	collaborators.GET("", collaboratorHandler.List)
	collaborators.POST("", collaboratorHandler.Add)
	collaborators.PUT("/:user_id", collaboratorHandler.Update)
	collaborators.DELETE("/:user_id", collaboratorHandler.Remove)

	// Share link routes (management)
	shares := canvases.Group("/:id/shares")
	shares.GET("", shareLinkHandler.List)
	shares.POST("", shareLinkHandler.Create)
	shares.DELETE("/:share_id", shareLinkHandler.Delete)

	// Library routes
	libraries := api.Group("/libraries")
	libraries.Use(middleware.AuthRequired(cfg))
	libraries.GET("", libraryHandler.List)
	libraries.POST("", libraryHandler.Create)
	libraries.GET("/:id", libraryHandler.Get)
	libraries.PUT("/:id", libraryHandler.Update)
	libraries.DELETE("/:id", libraryHandler.Delete)

	// Public share routes
	publicShares := api.Group("/shares")
	publicShares.GET("/:code", shareLinkHandler.GetByCode)
	publicShares.POST("/:code/validate", shareLinkHandler.Validate)

	// WebSocket
	wsAccessChecker := func(canvasID, userID, shareToken string) (string, error) {
		// Check share token first (access via share link)
		if shareToken != "" {
			claims, err := jwtpkg.ValidateShareToken(shareToken, cfg.JWTSecret)
			if err == nil && claims.CanvasID == canvasID {
				return claims.Permission, nil
			}
		}

		// Check if user is the canvas owner
		canvas, err := canvasRepo.GetByID(context.Background(), canvasID)
		if err != nil {
			return "", service.ErrCanvasNotFound
		}
		if canvas.OwnerID == userID {
			return "collaborate", nil
		}

		// Check collaborators table
		perm, err := collaboratorRepo.GetPermission(context.Background(), canvasID, userID)
		if err == nil {
			return perm, nil
		}

		return "", service.ErrCanvasAccessDenied
	}
	saveHandler := func(canvasID, userID string, data json.RawMessage) error {
		return canvasService.SaveData(context.Background(), canvasID, userID, data, int64(len(data)))
	}
	wsHandler := ws.NewHandler(wsHub, cfg, wsAccessChecker, saveHandler)
	r.GET("/ws", wsHandler.Upgrade)

	return r
}
