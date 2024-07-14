package apis

import (
	"strconv"
	"webp_server_go/pkg/api_services/models"

	"github.com/gofiber/fiber/v2"
)

type SiteHandler struct {
	repo models.SiteRepository
}

func NewSiteHandler(repo models.SiteRepository) *SiteHandler {
	return &SiteHandler{repo: repo}
}

// Get all sites
func (h *SiteHandler) GetSites(c *fiber.Ctx) error {
	sites, err := h.repo.GetAll()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(sites)
}

// Get a site by ID
func (h *SiteHandler) GetSite(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}
	site, err := h.repo.GetByID(uint(id))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Site not found",
		})
	}
	return c.JSON(site)
}

// Create a new site
func (h *SiteHandler) CreateSite(c *fiber.Ctx) error {
	var site models.Sites
	if err := c.BodyParser(&site); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}
	err := h.repo.Create(&site)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.Status(fiber.StatusCreated).JSON(site)
}

// Update a site
func (h *SiteHandler) UpdateSite(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}
	var site models.Sites
	if err := c.BodyParser(&site); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Cannot parse JSON",
		})
	}
	err = h.repo.Update(uint(id), &site)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.JSON(site)
}

// Delete a site
func (h *SiteHandler) DeleteSite(c *fiber.Ctx) error {
	id, err := strconv.ParseUint(c.Params("id"), 10, 64)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid ID",
		})
	}
	err = h.repo.Delete(uint(id))
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
