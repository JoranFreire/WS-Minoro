package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/ws-minoro/link-admin/internal/repository"
)

type fakeLinkRepo struct {
	mu           sync.Mutex
	links        map[uuid.UUID]*repository.Link
	destinations map[uuid.UUID][]repository.Destination
	createErr    error
}

func newFakeLinkRepo() *fakeLinkRepo {
	return &fakeLinkRepo{
		links:        map[uuid.UUID]*repository.Link{},
		destinations: map[uuid.UUID][]repository.Destination{},
	}
}

func (f *fakeLinkRepo) ListLinks(ctx context.Context, tenantID uuid.UUID) ([]repository.Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []repository.Link
	for _, l := range f.links {
		if l.TenantID == tenantID {
			out = append(out, *l)
		}
	}
	return out, nil
}

func (f *fakeLinkRepo) GetLinkByID(ctx context.Context, id, tenantID uuid.UUID) (*repository.Link, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.links[id]
	if !ok || l.TenantID != tenantID {
		return nil, errors.New("not found")
	}
	cp := *l
	return &cp, nil
}

func (f *fakeLinkRepo) CreateLink(ctx context.Context, l *repository.Link) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.createErr != nil {
		return f.createErr
	}
	l.ID = uuid.New()
	f.links[l.ID] = l
	return nil
}

func (f *fakeLinkRepo) UpdateLink(ctx context.Context, l *repository.Link) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	existing, ok := f.links[l.ID]
	if !ok || existing.TenantID != l.TenantID {
		return errors.New("not found")
	}
	f.links[l.ID] = l
	return nil
}

func (f *fakeLinkRepo) DeleteLink(ctx context.Context, id, tenantID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	l, ok := f.links[id]
	if !ok || l.TenantID != tenantID {
		return errors.New("not found")
	}
	delete(f.links, id)
	return nil
}

func (f *fakeLinkRepo) ListDestinations(ctx context.Context, linkID uuid.UUID) ([]repository.Destination, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.destinations[linkID], nil
}

func (f *fakeLinkRepo) CreateDestination(ctx context.Context, d *repository.Destination) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	d.ID = uuid.New()
	f.destinations[d.LinkID] = append(f.destinations[d.LinkID], *d)
	return nil
}

func (f *fakeLinkRepo) UpdateDestination(ctx context.Context, d *repository.Destination) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	dests := f.destinations[d.LinkID]
	for i, existing := range dests {
		if existing.ID == d.ID {
			dests[i] = *d
			return nil
		}
	}
	return errors.New("not found")
}

func (f *fakeLinkRepo) DeleteDestination(ctx context.Context, destID, linkID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	dests := f.destinations[linkID]
	for i, existing := range dests {
		if existing.ID == destID {
			f.destinations[linkID] = append(dests[:i], dests[i+1:]...)
			return nil
		}
	}
	return errors.New("not found")
}

func TestLinkService_Create_GeneratesShortCodeAndPersists(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	tenantID := uuid.New()

	link, err := svc.Create(context.Background(), tenantID, "My Link", "https://fallback.example.com", "round_robin")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if link.ShortCode == "" {
		t.Fatal("expected a generated short code")
	}
	if link.TenantID != tenantID {
		t.Fatalf("expected tenant %s, got %s", tenantID, link.TenantID)
	}
	if !link.IsActive {
		t.Fatal("expected new link to be active by default")
	}
}

func TestLinkService_Get_ScopesToTenant(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	tenantA := uuid.New()
	tenantB := uuid.New()

	created, err := svc.Create(context.Background(), tenantA, "Link A", "", "single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, _, err := svc.Get(context.Background(), created.ID, tenantB); err == nil {
		t.Fatal("expected error when fetching a link under the wrong tenant")
	}

	got, _, err := svc.Get(context.Background(), created.ID, tenantA)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected link %s, got %s", created.ID, got.ID)
	}
}

func TestLinkService_Update_ChangesFields(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	tenantID := uuid.New()

	created, err := svc.Create(context.Background(), tenantID, "Old Title", "", "single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	updated, err := svc.Update(context.Background(), created.ID, tenantID, "New Title", "https://new.example.com", "weighted", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "New Title" || updated.RoutingStrategy != "weighted" || updated.IsActive {
		t.Fatalf("unexpected updated link: %+v", updated)
	}
}

func TestLinkService_Delete_RemovesLink(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	tenantID := uuid.New()

	created, err := svc.Create(context.Background(), tenantID, "Title", "", "single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.Delete(context.Background(), created.ID, tenantID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, _, err := svc.Get(context.Background(), created.ID, tenantID); err == nil {
		t.Fatal("expected link to be gone after delete")
	}
}

func TestLinkService_AddDestination_DefaultsToActive(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	linkID := uuid.New()
	maxClicks := 100

	dest, err := svc.AddDestination(context.Background(), linkID, "https://a.example.com", 3, &maxClicks)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !dest.IsActive {
		t.Fatal("expected new destination to default to active")
	}
	if dest.Weight != 3 || *dest.MaxClicks != 100 {
		t.Fatalf("unexpected destination: %+v", dest)
	}
}

func TestLinkService_DeleteDestination_RemovesIt(t *testing.T) {
	repo := newFakeLinkRepo()
	svc := NewLinkService(repo)
	linkID := uuid.New()

	dest, err := svc.AddDestination(context.Background(), linkID, "https://a.example.com", 1, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if err := svc.DeleteDestination(context.Background(), dest.ID, linkID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dests, err := repo.ListDestinations(context.Background(), linkID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(dests) != 0 {
		t.Fatalf("expected destination to be removed, got %d remaining", len(dests))
	}
}
