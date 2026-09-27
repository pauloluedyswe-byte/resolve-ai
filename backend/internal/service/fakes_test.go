package service

import (
	"context"
	"io"
	"sync"
	"time"

	"resolveai/internal/domain"
)

type fakeUsers struct {
	mu    sync.Mutex
	users map[int64]*domain.User
	next  int64
}

func newFakeUsers() *fakeUsers { return &fakeUsers{users: map[int64]*domain.User{}} }

func (f *fakeUsers) Create(_ context.Context, u *domain.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	u.ID, u.CreatedAt = f.next, time.Now()
	cp := *u
	f.users[u.ID] = &cp
	return nil
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			cp := *u
			return &cp, nil
		}
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) GetByID(_ context.Context, id int64) (*domain.User, error) {
	if u, ok := f.users[id]; ok {
		cp := *u
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeUsers) ListByRole(_ context.Context, role domain.Role) ([]domain.User, error) {
	var out []domain.User
	for _, u := range f.users {
		if u.Role == role {
			out = append(out, *u)
		}
	}
	return out, nil
}

type fakeCategories struct{}

func (fakeCategories) List(context.Context) ([]domain.Category, error) {
	return []domain.Category{{ID: 1, Name: "Iluminação"}}, nil
}
func (fakeCategories) Exists(_ context.Context, id int64) (bool, error) { return id == 1, nil }

type fakeOccurrences struct {
	occ      map[int64]*domain.Occurrence
	comments []domain.Comment
	history  []domain.StatusChange
	next     int64
}

func newFakeOccurrences() *fakeOccurrences {
	return &fakeOccurrences{occ: map[int64]*domain.Occurrence{}}
}

func (f *fakeOccurrences) Create(_ context.Context, o *domain.Occurrence) error {
	f.next++
	o.ID = f.next
	cp := *o
	f.occ[o.ID] = &cp
	f.history = append(f.history, domain.StatusChange{OccurrenceID: o.ID, ToStatus: o.Status, ChangedBy: o.RequesterID})
	return nil
}

func (f *fakeOccurrences) GetByID(_ context.Context, id int64) (*domain.Occurrence, error) {
	if o, ok := f.occ[id]; ok {
		cp := *o
		return &cp, nil
	}
	return nil, domain.ErrNotFound
}

func (f *fakeOccurrences) List(_ context.Context, flt domain.OccurrenceFilter) ([]domain.Occurrence, error) {
	var out []domain.Occurrence
	for _, o := range f.occ {
		if flt.RequesterID != nil && o.RequesterID != *flt.RequesterID {
			continue
		}
		out = append(out, *o)
	}
	return out, nil
}

func (f *fakeOccurrences) ChangeStatus(_ context.Context, id int64, from, to domain.Status, userID int64, note string, solution *string) error {
	o := f.occ[id]
	if o.Status != from {
		return domain.ErrConflict
	}
	o.Status = to
	if solution != nil {
		o.Solution = solution
	}
	from2 := from
	f.history = append(f.history, domain.StatusChange{OccurrenceID: id, FromStatus: &from2, ToStatus: to, ChangedBy: userID, Note: note})
	return nil
}

func (f *fakeOccurrences) UpdatePriority(_ context.Context, id int64, p domain.Priority) error {
	f.occ[id].Priority = p
	return nil
}

func (f *fakeOccurrences) UpdateAssignee(_ context.Context, id int64, a *int64) error {
	f.occ[id].AssigneeID = a
	return nil
}

func (f *fakeOccurrences) UpdateSolution(_ context.Context, id int64, s string) error {
	f.occ[id].Solution = &s
	return nil
}

func (f *fakeOccurrences) UpdateImage(_ context.Context, id int64, url string) error {
	f.occ[id].ImageURL = &url
	return nil
}

func (f *fakeOccurrences) SetRating(_ context.Context, id int64, r int, c string) error {
	f.occ[id].Rating = &r
	f.occ[id].RatingComment = &c
	return nil
}

func (f *fakeOccurrences) AddComment(_ context.Context, c *domain.Comment) error {
	c.ID = int64(len(f.comments) + 1)
	f.comments = append(f.comments, *c)
	return nil
}

func (f *fakeOccurrences) ListComments(_ context.Context, id int64) ([]domain.Comment, error) {
	var out []domain.Comment
	for _, c := range f.comments {
		if c.OccurrenceID == id {
			out = append(out, c)
		}
	}
	return out, nil
}

func (f *fakeOccurrences) ListHistory(_ context.Context, id int64) ([]domain.StatusChange, error) {
	var out []domain.StatusChange
	for _, h := range f.history {
		if h.OccurrenceID == id {
			out = append(out, h)
		}
	}
	return out, nil
}

func (f *fakeOccurrences) Stats(context.Context) (*domain.DashboardStats, error) {
	return &domain.DashboardStats{Total: len(f.occ)}, nil
}

type fakeFiles struct{}

func (fakeFiles) Save(_ context.Context, _, ext string, r io.Reader) (string, error) {
	io.Copy(io.Discard, r)
	return "/uploads/test" + ext, nil
}
