package apis

import (
	"github.com/gofiber/fiber/v2"
	"webp_server_go/pkg/api_services/database"
	"webp_server_go/pkg/api_services/models"
)

func SetupRoutes(app *fiber.App, db *database.Database, allowedKey string) {
	repo := models.NewSiteRepository(db)
	handler := NewSiteHandler(repo)

	api := app.Group("/api")

	api.Use(APIKeyAuth(allowedKey)) // Middleware to check API key

	api.Get("/sites", handler.GetSites)
	api.Get("/sites/:id", handler.GetSite)
	api.Post("/sites", handler.CreateSite)
	api.Put("/sites/:id", handler.UpdateSite)
	api.Delete("/sites/:id", handler.DeleteSite)
}
