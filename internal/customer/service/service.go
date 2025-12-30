package service

import (
	"context"
	"fmt"
	"regexp"

	"testovoe/internal/customer/repo"
)

type Service struct {
	repo *repo.Repository
}

func NewService(repo *repo.Repository) *Service {
	return &Service{repo: repo}
}

var idnRegex = regexp.MustCompile(`^\d{12}$`)

func (s *Service) ValidateIDN(idn string) error {
	if !idnRegex.MatchString(idn) {
		return fmt.Errorf("idn must be exactly 12 digits")
	}
	return nil
}

func (s *Service) UpsertCustomer(ctx context.Context, idn string) (*repo.Customer, error) {
	if err := s.ValidateIDN(idn); err != nil {
		return nil, err
	}

	customer, err := s.repo.UpsertCustomer(ctx, idn)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert customer: %w", err)
	}

	return customer, nil
}

func (s *Service) GetCustomer(ctx context.Context, idn string) (*repo.Customer, error) {
	if err := s.ValidateIDN(idn); err != nil {
		return nil, err
	}

	customer, err := s.repo.GetCustomer(ctx, idn)
	if err != nil {
		return nil, fmt.Errorf("failed to get customer: %w", err)
	}

	return customer, nil
}

