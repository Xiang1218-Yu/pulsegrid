package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pulsegrid/internal/analytics"
	"pulsegrid/internal/domain"
	"pulsegrid/internal/events"
	"pulsegrid/internal/jobs"
	"pulsegrid/internal/service"
	"pulsegrid/internal/store"
)

type Config struct {
	App     *service.App
	Events  *events.Bus
	Jobs    *jobs.Queue
	Metrics *analytics.Store
	Logger  *slog.Logger
}

type Server struct {
	config Config
	mux    *http.ServeMux
}

func New(config Config) *Server {
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	server := &Server{config: config, mux: http.NewServeMux()}
	server.routes()
	return server
}

func (s *Server) Handler() http.Handler {
	return withLogging(s.config.Logger, withRecovery(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /healthz", s.health)
	s.mux.HandleFunc("GET /readyz", s.ready)
	s.mux.HandleFunc("GET /v1/overview", s.overview)
	s.mux.HandleFunc("GET /v1/metrics", s.metrics)
	s.mux.HandleFunc("GET /v1/metrics/daily", s.metricsDaily)
	s.mux.HandleFunc("GET /v1/events", s.events)
	s.mux.HandleFunc("GET /v1/export/{organization_id}", s.exportOrganization)

	s.mux.HandleFunc("GET /v1/organizations", s.organizationsList)
	s.mux.HandleFunc("POST /v1/organizations", s.organizationsCreate)
	s.mux.HandleFunc("GET /v1/organizations/{id}", s.organizationsGet)
	s.mux.HandleFunc("PUT /v1/organizations/{id}", s.organizationsUpdate)
	s.mux.HandleFunc("POST /v1/organizations/{id}/suspend", s.organizationsSuspend)
	s.mux.HandleFunc("POST /v1/organizations/{id}/resume", s.organizationsResume)
	s.mux.HandleFunc("POST /v1/organizations/{id}/archive", s.organizationsArchive)

	s.mux.HandleFunc("GET /v1/contacts", s.contactsList)
	s.mux.HandleFunc("POST /v1/contacts", s.contactsCreate)
	s.mux.HandleFunc("GET /v1/contacts/{id}", s.contactsGet)
	s.mux.HandleFunc("PUT /v1/contacts/{id}", s.contactsUpdate)
	s.mux.HandleFunc("POST /v1/contacts/{id}/unsubscribe", s.contactsUnsubscribe)
	s.mux.HandleFunc("POST /v1/contacts/{id}/subscribe", s.contactsSubscribe)

	s.mux.HandleFunc("GET /v1/audiences", s.audiencesList)
	s.mux.HandleFunc("POST /v1/audiences", s.audiencesCreate)
	s.mux.HandleFunc("GET /v1/audiences/{id}", s.audiencesGet)
	s.mux.HandleFunc("PUT /v1/audiences/{id}", s.audiencesUpdate)
	s.mux.HandleFunc("GET /v1/audiences/{id}/resolve", s.audiencesResolve)

	s.mux.HandleFunc("GET /v1/templates", s.templatesList)
	s.mux.HandleFunc("POST /v1/templates", s.templatesCreate)
	s.mux.HandleFunc("GET /v1/templates/{id}", s.templatesGet)
	s.mux.HandleFunc("PUT /v1/templates/{id}", s.templatesUpdate)

	s.mux.HandleFunc("GET /v1/campaigns", s.campaignsList)
	s.mux.HandleFunc("POST /v1/campaigns", s.campaignsCreate)
	s.mux.HandleFunc("GET /v1/campaigns/{id}", s.campaignsGet)
	s.mux.HandleFunc("POST /v1/campaigns/{id}/schedule", s.campaignsSchedule)
	s.mux.HandleFunc("POST /v1/campaigns/{id}/start", s.campaignsStart)
	s.mux.HandleFunc("POST /v1/campaigns/{id}/pause", s.campaignsPause)
	s.mux.HandleFunc("POST /v1/campaigns/{id}/complete", s.campaignsComplete)
	s.mux.HandleFunc("POST /v1/campaigns/{id}/cancel", s.campaignsCancel)

	s.mux.HandleFunc("GET /v1/deliveries", s.deliveriesList)
	s.mux.HandleFunc("GET /v1/deliveries/{id}", s.deliveriesGet)
	s.mux.HandleFunc("POST /v1/deliveries/{id}/event", s.deliveriesEvent)

	s.mux.HandleFunc("GET /v1/automations", s.automationsList)
	s.mux.HandleFunc("POST /v1/automations", s.automationsCreate)
	s.mux.HandleFunc("GET /v1/automations/{id}", s.automationsGet)
	s.mux.HandleFunc("PUT /v1/automations/{id}", s.automationsUpdate)
	s.mux.HandleFunc("POST /v1/automations/process", s.automationsProcess)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "service": "pulsegrid", "time": time.Now().UTC()})
}

func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	if s.config.App == nil {
		writeError(w, errors.New("application is not configured"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ready": true})
}

func (s *Server) overview(w http.ResponseWriter, r *http.Request) {
	result, err := s.config.App.Overview(r.Context(), service.OverviewOptions{OrganizationID: r.URL.Query().Get("organization_id")})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	if s.config.Metrics == nil {
		writeError(w, errors.New("metrics are not configured"))
		return
	}
	rows := s.config.Metrics.Summarize(analytics.Filter{OrganizationID: r.URL.Query().Get("organization_id"), Name: r.URL.Query().Get("name"), Limit: parseInt(r.URL.Query().Get("limit"), 1000)})
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) metricsDaily(w http.ResponseWriter, r *http.Request) {
	if s.config.Metrics == nil {
		writeError(w, errors.New("metrics are not configured"))
		return
	}
	rows := s.config.Metrics.Daily(analytics.Filter{OrganizationID: r.URL.Query().Get("organization_id"), Name: r.URL.Query().Get("name"), Limit: parseInt(r.URL.Query().Get("limit"), 1000)})
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) events(w http.ResponseWriter, r *http.Request) {
	if s.config.Events == nil {
		writeError(w, errors.New("events are not configured"))
		return
	}
	subscription := s.config.Events.Subscribe(r.URL.Query().Get("topic"))
	defer subscription.Close()
	go func() {
		<-r.Context().Done()
		subscription.Close()
	}()
	w.Header().Set("content-type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	flusher, _ := w.(http.Flusher)
	encoder := json.NewEncoder(w)
	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-subscription.Events():
			if !ok {
				return
			}
			if err := encoder.Encode(event); err != nil {
				return
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func (s *Server) exportOrganization(w http.ResponseWriter, r *http.Request) {
	data, err := s.config.App.ExportOrganization(r.Context(), r.PathValue("organization_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("content-disposition", "attachment; filename=organization.json")
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (s *Server) organizationsList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.config.App.ListOrganizations(r.Context(), store.OrganizationFilter{
		Status: query.Get("status"), Owner: query.Get("owner"), Search: query.Get("search"), Tag: query.Get("tag"), Limit: parseInt(query.Get("limit"), 100),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) organizationsCreate(w http.ResponseWriter, r *http.Request) {
	var input service.OrganizationInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateOrganization(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) organizationsGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) organizationsUpdate(w http.ResponseWriter, r *http.Request) {
	var input service.OrganizationInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.UpdateOrganization(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) organizationsSuspend(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.SuspendOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) organizationsResume(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.ResumeOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) organizationsArchive(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.ArchiveOrganization(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) contactsList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.config.App.ListContacts(r.Context(), store.ContactFilter{
		OrganizationID: query.Get("organization_id"), Status: query.Get("status"), Search: query.Get("search"), Tag: query.Get("tag"), Limit: parseInt(query.Get("limit"), 100),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) contactsCreate(w http.ResponseWriter, r *http.Request) {
	var input service.ContactInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateContact(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) contactsGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetContact(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) contactsUpdate(w http.ResponseWriter, r *http.Request) {
	var input service.ContactInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.UpdateContact(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) contactsUnsubscribe(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.UnsubscribeContact(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) contactsSubscribe(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.SubscribeContact(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) audiencesList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.config.App.ListAudiences(r.Context(), r.URL.Query().Get("organization_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) audiencesCreate(w http.ResponseWriter, r *http.Request) {
	var input service.AudienceInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateAudience(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) audiencesGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetAudience(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) audiencesUpdate(w http.ResponseWriter, r *http.Request) {
	var input service.AudienceInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.UpdateAudience(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) audiencesResolve(w http.ResponseWriter, r *http.Request) {
	rows, err := s.config.App.ResolveAudience(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) templatesList(w http.ResponseWriter, r *http.Request) {
	rows, err := s.config.App.ListTemplates(r.Context(), r.URL.Query().Get("organization_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) templatesCreate(w http.ResponseWriter, r *http.Request) {
	var input service.TemplateInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateTemplate(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) templatesGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetTemplate(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) templatesUpdate(w http.ResponseWriter, r *http.Request) {
	var input service.TemplateInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.UpdateTemplate(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.config.App.ListCampaigns(r.Context(), store.CampaignFilter{
		OrganizationID: query.Get("organization_id"), Status: query.Get("status"), Search: query.Get("search"), Limit: parseInt(query.Get("limit"), 100),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) campaignsCreate(w http.ResponseWriter, r *http.Request) {
	var input service.CampaignInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateCampaign(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) campaignsGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsSchedule(w http.ResponseWriter, r *http.Request) {
	var input struct {
		ScheduledAt time.Time `json:"scheduled_at"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.ScheduleCampaign(r.Context(), r.PathValue("id"), input.ScheduledAt)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsStart(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.StartCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsPause(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.PauseCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsComplete(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.CompleteCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) campaignsCancel(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.CancelCampaign(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) deliveriesList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.config.App.ListDeliveries(r.Context(), store.DeliveryFilter{
		CampaignID: query.Get("campaign_id"), ContactID: query.Get("contact_id"), Status: query.Get("status"), Limit: parseInt(query.Get("limit"), 100),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) deliveriesGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetDelivery(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) deliveriesEvent(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Status domain.MessageStatus `json:"status"`
	}
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.RegisterDeliveryEvent(r.Context(), r.PathValue("id"), input.Status)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) automationsList(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	rows, err := s.config.App.ListAutomations(r.Context(), store.AutomationFilter{
		OrganizationID: query.Get("organization_id"), Trigger: query.Get("trigger"), Status: query.Get("status"), Limit: parseInt(query.Get("limit"), 100),
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, collection(rows))
}

func (s *Server) automationsCreate(w http.ResponseWriter, r *http.Request) {
	var input service.AutomationInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.CreateAutomation(r.Context(), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, value)
}

func (s *Server) automationsGet(w http.ResponseWriter, r *http.Request) {
	value, err := s.config.App.GetAutomation(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) automationsUpdate(w http.ResponseWriter, r *http.Request) {
	var input service.AutomationInput
	if !decode(w, r, &input) {
		return
	}
	value, err := s.config.App.UpdateAutomation(r.Context(), r.PathValue("id"), input)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, value)
}

func (s *Server) automationsProcess(w http.ResponseWriter, r *http.Request) {
	var event domain.Event
	if !decode(w, r, &event) {
		return
	}
	if err := s.config.App.ProcessEvent(r.Context(), event); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]any{"processed": true, "event": event})
}

func decode(w http.ResponseWriter, r *http.Request, target any) bool {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("content-type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, domain.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists), errors.Is(err, domain.ErrConflict), errors.Is(err, domain.ErrInvalidState):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	}
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func parseInt(value string, fallback int) int {
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func collection[T any](items []T) map[string]any {
	return map[string]any{"items": items, "count": len(items)}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
func (w *statusWriter) Write(value []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(value)
}

func withLogging(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		writer := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(writer, r)
		logger.Info("http request", "method", r.Method, "path", r.URL.Path, "status", writer.status, "duration", time.Since(start))
	})
}

func withRecovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if recover() != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func normalizeHeader(value string) string { return strings.TrimSpace(value) }
