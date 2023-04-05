package controller

import (
	"context"

	"github.com/Onelvay/docker-compose-project/pkg/domain"
	"gorm.io/gorm"
)

type TokenPostgres struct {
	Db *gorm.DB
}

func NewTokenDbController(db *gorm.DB) *BookstorePostgres {
	return &BookstorePostgres{Db: db}
}

func (r *BookstorePostgres) CreateToken(cnt context.Context, token domain.Refresh_token) bool {
	r.Db.Create(&token)
	return true
}
func (r *BookstorePostgres) GetToken(cxt context.Context, token string) domain.Refresh_token {
	var d domain.Refresh_token
	r.Db.Where("token= ?", token).Find(&d)
	r.Db.Delete(&d)
	return d
}
