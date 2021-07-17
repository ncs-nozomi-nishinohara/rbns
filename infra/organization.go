package infra

import (
	"github.com/n-creativesystem/rbnc/domain/model"
	"github.com/n-creativesystem/rbnc/domain/repository"
	"github.com/n-creativesystem/rbnc/infra/entity"
	"gorm.io/gorm"
)

type organization struct {
	master *gorm.DB
	slave  *gorm.DB
}

var (
	_ repository.Organization        = (*organization)(nil)
	_ repository.OrganizationCommand = (*organization)(nil)
)

func (r *organization) FindAll() (model.Organizations, error) {
	session := r.slave
	var organizations []entity.Organization
	err := session.Order("id").Preload("Users").Find(&organizations).Error
	if err != nil {
		return nil, model.NewDBErr(err)
	}
	mOrganizations := make(model.Organizations, len(organizations))
	for i, organization := range organizations {
		if o, err := organization.ConvertModel(); err != nil {
			return nil, err
		} else {
			mOrganizations[i] = *o
		}
	}
	return mOrganizations, nil
}

func (r *organization) FindByID(id model.ID) (*model.Organization, error) {
	session := r.slave
	var organization entity.Organization
	err := session.Order("id").Preload("Users").Where(&entity.Organization{Model: entity.Model{ID: *id.Value()}}).Find(&organization).Error
	if err != nil {
		return nil, model.NewDBErr(err)
	}
	return organization.ConvertModel()
}

func (r *organization) FindByName(name model.Name) (*model.Organization, error) {
	var org entity.Organization
	if err := r.slave.Where(&entity.Organization{Name: *name.Value()}).Find(&org).Error; err != nil {
		return nil, model.NewDBErr(err)
	}
	return org.ConvertModel()
}

func (r *organization) Create(name model.Name, description string) (*model.Organization, error) {
	entity := entity.Organization{
		Name:        *name.Value(),
		Description: description,
	}
	entity.Generate()
	err := r.master.Create(&entity).Error
	if err != nil {
		return nil, model.NewDBErr(err)
	}
	return model.NewOrganization(entity.ID, entity.Name, entity.Description)
}

func (r *organization) Update(organization *model.Organization) error {
	value := entity.Organization{
		Name:        *organization.GetName(),
		Description: organization.GetDescription(),
	}
	return model.NewDBErr(r.master.Where(&entity.Organization{Model: entity.Model{ID: *organization.GetID()}}).Updates(&value).Error)
}

func (r *organization) Delete(id model.ID) error {
	db := r.master.Where(&entity.Organization{Model: entity.Model{ID: *id.Value()}}).Delete(&entity.Organization{})
	if db.RowsAffected == 0 {
		return model.ErrNoData
	}
	return model.NewDBErr(db.Error)
}
