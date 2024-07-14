package models

import "webp_server_go/pkg/api_services/database"

type SiteRepository interface {
	GetAll() ([]Sites, error)
	GetByID(id uint) (Sites, error)
	Create(site *Sites) error
	Update(id uint, site *Sites) error
	Delete(id uint) error
}
type siteRepository struct {
	db *database.Database
}

func NewSiteRepository(db *database.Database) SiteRepository {
	return &siteRepository{db}
}

func (r *siteRepository) GetAll() ([]Sites, error) {
	var sites []Sites
	result := r.db.DB.Find(&sites)
	if result.Error != nil {
		return nil, result.Error
	}
	return sites, nil
}

func (r *siteRepository) GetByID(id uint) (Sites, error) {
	var site Sites
	result := r.db.DB.First(&site, id)
	if result.Error != nil {
		return Sites{}, result.Error
	}
	return site, nil
}

func (r *siteRepository) Create(site *Sites) error {
	result := r.db.DB.Create(site)
	return result.Error
}

func (r *siteRepository) Update(id uint, site *Sites) error {
	existingSite, err := r.GetByID(id)
	if err != nil {
		return err
	}

	existingSite.Title = site.Title
	existingSite.Domain = site.Domain
	existingSite.AllowedTypes = site.AllowedTypes
	existingSite.Origin.URL = site.Origin.URL
	existingSite.Origin.S3Config = site.Origin.S3Config

	result := r.db.DB.Save(&existingSite)
	return result.Error
}

func (r *siteRepository) Delete(id uint) error {
	result := r.db.DB.Delete(&Sites{}, id)
	return result.Error
}
