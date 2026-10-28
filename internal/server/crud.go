package server

import (
	"context"
	"encoding/json"
	"net/http"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"

	"github.com/go-chi/chi/v5"
)

// crudOpts customizes a generic document resource.
type crudOpts[T any] struct {
	entity    string
	prepare   func(*T) error
	check     func(*http.Request, *T) error // validation needing the request (tenant references)
	reload    bool
	readPerm  domain.Permission // "" = any signed-in user of the tenant
	writePerm domain.Permission // required; defaults to settings.manage
	history   bool              // keep a revision of every change
	onDelete  func(r *http.Request, id int64)
	filter    func(items []T, q string) []T
}

// mountCRUD registers list/create/get/update/delete routes for a collection.
func mountCRUD[T any](r chi.Router, path string, s *Server, c *store.Collection[T], o crudOpts[T]) {
	if o.writePerm == "" {
		o.writePerm = domain.PermSettingsManage
	}
	read := func(next http.Handler) http.Handler { return next }
	if o.readPerm != "" {
		read = requirePerm(o.readPerm)
	}
	write := requirePerm(o.writePerm)
	changed := func(r *http.Request, action string, id int64) {
		s.audit(r, action, o.entity, id, "")
		if o.history {
			s.revision(r, c, action, id)
		}
		if o.reload {
			s.configChanged(r, o.entity)
		}
	}
	r.With(read).Get(path, handler(func(w http.ResponseWriter, r *http.Request) error {
		items, err := c.List(r.Context())
		if err != nil {
			return err
		}
		if q := r.URL.Query().Get("q"); q != "" && o.filter != nil {
			items = o.filter(items, q)
		}
		writeJSON(w, http.StatusOK, items)
		return nil
	}).ServeHTTP)
	r.With(read).Get(path+"/{id}", handler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		item, err := c.Get(r.Context(), id)
		if err != nil {
			return err
		}
		writeJSON(w, http.StatusOK, item)
		return nil
	}).ServeHTTP)
	r.With(write).Post(path, handler(func(w http.ResponseWriter, r *http.Request) error {
		var item T
		if err := decode(r, &item); err != nil {
			return err
		}
		if err := o.validate(r, &item); err != nil {
			return err
		}
		if err := c.Create(r.Context(), &item); err != nil {
			return err
		}
		changed(r, "create", c.IDOf(&item))
		writeJSON(w, http.StatusCreated, item)
		return nil
	}).ServeHTTP)
	r.With(write).Put(path+"/{id}", handler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		var item T
		if err := decode(r, &item); err != nil {
			return err
		}
		if _, err := c.Get(r.Context(), id); err != nil {
			return err // unknown here, including other tenants' objects
		}
		c.SetID(&item, id)
		if err := o.validate(r, &item); err != nil {
			return err
		}
		if err := c.Update(r.Context(), &item); err != nil {
			return err
		}
		changed(r, "update", id)
		writeJSON(w, http.StatusOK, item)
		return nil
	}).ServeHTTP)
	r.With(write).Delete(path+"/{id}", handler(func(w http.ResponseWriter, r *http.Request) error {
		id, err := pathID(r)
		if err != nil {
			return err
		}
		if err := c.Delete(r.Context(), id); err != nil {
			return err
		}
		if o.onDelete != nil {
			o.onDelete(r, id)
		}
		changed(r, "delete", id)
		ok(w)
		return nil
	}).ServeHTTP)
}

// revision snapshots the current state of a document into its history.
func (s *Server) revision(r *http.Request, c interface {
	Kind() string
}, action string, id int64) {
	var data any = map[string]any{"deleted": true}
	if getter, ok := c.(interface {
		GetJSON(ctx context.Context, id int64) (string, error)
	}); ok && action != "delete" {
		if raw, err := getter.GetJSON(r.Context(), id); err == nil {
			data = json.RawMessage(raw)
		}
	}
	raw, _ := json.Marshal(data)
	p := principalOf(r)
	_ = s.d.Store.AddRevision(r.Context(), domain.Revision{Entity: c.Kind(), EntityID: id, Action: action,
		Data: string(raw), ChangedBy: p.user.ID, ChangedByName: p.user.DisplayName})
}

func (o crudOpts[T]) validate(r *http.Request, item *T) error {
	if o.prepare != nil {
		if err := o.prepare(item); err != nil {
			return err
		}
	}
	if o.check != nil {
		return o.check(r, item)
	}
	return nil
}
