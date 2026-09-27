package httpapi

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"resolveai/internal/domain"
	"resolveai/internal/service"
)

const maxImageSize = 5 << 20 // 5 MB

// ---- Arquivos ----

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	contentType, data, err := s.files.Get(r.Context(), chi.URLParam(r, "name"))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}

// ---- Autenticação ----

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var in struct{ Name, Email, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	res, err := s.auth.Register(r.Context(), in.Name, in.Email, in.Password)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, res)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !decodeJSON(w, r, &in) {
		return
	}
	res, err := s.auth.Login(r.Context(), in.Email, in.Password)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	u, err := s.auth.Me(r.Context(), actorFrom(r).ID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) listGestores(w http.ResponseWriter, r *http.Request) {
	users, err := s.auth.ListGestores(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// ---- Ocorrências ----

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	cats, err := s.occurrences.Categories(r.Context())
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, cats)
}

func (s *Server) listOccurrences(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := domain.OccurrenceFilter{Search: q.Get("q")}
	if v := q.Get("category_id"); v != "" {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "category_id inválido")
			return
		}
		f.CategoryID = &id
	}
	if v := domain.Status(q.Get("status")); v != "" {
		if !v.Valid() {
			writeError(w, http.StatusBadRequest, "status inválido")
			return
		}
		f.Status = &v
	}
	if v := domain.Priority(q.Get("priority")); v != "" {
		if !v.Valid() {
			writeError(w, http.StatusBadRequest, "prioridade inválida")
			return
		}
		f.Priority = &v
	}
	list, err := s.occurrences.List(r.Context(), actorFrom(r), f)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func (s *Server) createOccurrence(w http.ResponseWriter, r *http.Request) {
	var in service.CreateOccurrenceInput
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.Create(r.Context(), actorFrom(r), in)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func (s *Server) getOccurrence(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	d, err := s.occurrences.Get(r.Context(), actorFrom(r), id)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, d)
}

func (s *Server) uploadImage(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxImageSize+1024)
	file, header, err := r.FormFile("image")
	if err != nil {
		writeError(w, http.StatusBadRequest, "envie a imagem no campo 'image' (máx. 5 MB)")
		return
	}
	defer file.Close()
	o, err := s.occurrences.AttachImage(r.Context(), actorFrom(r), id, header.Header.Get("Content-Type"), file)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) addComment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct{ Body string }
	if !decodeJSON(w, r, &in) {
		return
	}
	c, err := s.occurrences.AddComment(r.Context(), actorFrom(r), id, in.Body)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) changeStatus(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in service.ChangeStatusInput
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.ChangeStatus(r.Context(), actorFrom(r), id, in)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) rate(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Rating  int    `json:"rating"`
		Comment string `json:"comment"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.Rate(r.Context(), actorFrom(r), id, in.Rating, in.Comment)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) updatePriority(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Priority domain.Priority `json:"priority"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.UpdatePriority(r.Context(), actorFrom(r), id, in.Priority)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) assign(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		AssigneeID *int64 `json:"assignee_id"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.Assign(r.Context(), actorFrom(r), id, in.AssigneeID)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) registerSolution(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r)
	if !ok {
		return
	}
	var in struct {
		Solution string `json:"solution"`
	}
	if !decodeJSON(w, r, &in) {
		return
	}
	o, err := s.occurrences.RegisterSolution(r.Context(), actorFrom(r), id, in.Solution)
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	stats, err := s.occurrences.Dashboard(r.Context(), actorFrom(r))
	if err != nil {
		writeDomainError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}
