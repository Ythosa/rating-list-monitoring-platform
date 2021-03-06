package dto

import (
	"fmt"

	"github.com/go-playground/validator/v10"

	"github.com/ythosa/rating-list-monitoring-platform-api/pkg/validation"
)

type SigningUp struct {
	Username   string `json:"username" validate:"required,min=4,max=10"`
	Password   string `json:"password" validate:"required,min=5,max=20"`
	FirstName  string `json:"first_name" validate:"required,alpha,min=3"`
	MiddleName string `json:"middle_name" validate:"required,alpha,min=3"`
	LastName   string `json:"last_name" validate:"required,alpha,min=3"`
	Snils      string `json:"snils" validate:"required,numeric,len=11"`
}

func (d *SigningUp) Validate(validate *validator.Validate) error {
	if err := validate.Struct(d); err != nil {
		return fmt.Errorf("failed to validate dto: %w", err)
	}

	if err := validation.Snils(d.Snils); err != nil {
		return fmt.Errorf("invalid snils: %w", err)
	}

	return nil
}
