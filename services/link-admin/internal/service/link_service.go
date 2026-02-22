package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type LinkService struct {
	repo *repository.Repository
}

func NewLinkService(repo *repository.Repository) *LinkService {
	return &LinkService{repo: repo}
}

func (s *LinkService) List(ctx context.Context, tenantID uuid.UUID) ([]repository.Link, error) {
	return s.repo.ListLinks(ctx, tenantID)
}

func (s *LinkService) Get(ctx context.Context, id, tenantID uuid.UUID) (*repository.Link, []repository.Destination, error) {
	link, err := s.repo.GetLinkByID(ctx, id, tenantID)
	if err != nil {
		return nil, nil, err
	}
	dests, err := s.repo.ListDestinations(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return link, dests, nil
}

func (s *LinkService) Create(ctx context.Context, tenantID uuid.UUID, title, fallbackURL, strategy string) (*repository.Link, error) {
	shortCode, err := generateShortCode()
	if err != nil {
		return nil, err
	}
	link := &repository.Link{
		TenantID:        tenantID,
		ShortCode:       shortCode,
		Title:           title,
		FallbackURL:     fallbackURL,
		RoutingStrategy: strategy,
		IsActive:        true,
	}
	if err := s.repo.CreateLink(ctx, link); err != nil {
		return nil, err
	}
	return link, nil
}

func (s *LinkService) Update(ctx context.Context, id, tenantID uuid.UUID, title, fallbackURL, strategy string, isActive bool) (*repository.Link, error) {
	link, err := s.repo.GetLinkByID(ctx, id, tenantID)
	if err != nil {
		return nil, err
	}
	link.Title = title
	link.FallbackURL = fallbackURL
	link.RoutingStrategy = strategy
	link.IsActive = isActive
	return link, s.repo.UpdateLink(ctx, link)
}

func (s *LinkService) Delete(ctx context.Context, id, tenantID uuid.UUID) error {
	return s.repo.DeleteLink(ctx, id, tenantID)
}

func (s *LinkService) AddDestination(ctx context.Context, linkID uuid.UUID, url string, weight int, maxClicks *int) (*repository.Destination, error) {
	d := &repository.Destination{
		LinkID:    linkID,
		URL:       url,
		Weight:    weight,
		MaxClicks: maxClicks,
		IsActive:  true,
	}
	return d, s.repo.CreateDestination(ctx, d)
}

func (s *LinkService) UpdateDestination(ctx context.Context, destID, linkID uuid.UUID, url string, weight int, maxClicks *int, isActive bool) error {
	d := &repository.Destination{
		ID:        destID,
		LinkID:    linkID,
		URL:       url,
		Weight:    weight,
		MaxClicks: maxClicks,
		IsActive:  isActive,
	}
	return s.repo.UpdateDestination(ctx, d)
}

func (s *LinkService) DeleteDestination(ctx context.Context, destID, linkID uuid.UUID) error {
	return s.repo.DeleteDestination(ctx, destID, linkID)
}

func generateShortCode() (string, error) {
	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b)[:8], nil
}
