package server

import (
	"net/http"

	"callcenter-middleware/internal/domain"
	"callcenter-middleware/internal/store"
	"github.com/go-chi/chi/v5"
)

// crudOpts customizes a generic document resource.
type crudOpts[T any] struct {
	entity    string
	prepare   func(*T) error
	reload    bool
	readRole  domain.Role
	writeRole domain.Role
	onDelete  func(r *http.Request, id int64)
	filter    func(items []T, q string) []T
}

// mountCRUD registers list/create/get/update/delete routes for a collection.
func mountCRUD[T any](r chi.Router, path string, s *Server, c *store.Collection[T], o crudOpts[T]) {
	if o.readRole == "" {
		o.readRole = domain.RoleAgent
	}
	if o.writeRole == "" {
		o.writeRole = domain.RoleAdmin
	}
	read, write := require(o.readRole), require(o.writeRole)
	changed := func(r *http.Request, action string, id int64) {
		s.audit(r, action, o.entity, id, "")
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
		if o.prepare != nil {
			if err := o.prepare(&item); err != nil {
				return err
			}
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
		c.SetID(&item, id)
		if o.prepare != nil {
			if err := o.prepare(&item); err != nil {
				return err
			}
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
