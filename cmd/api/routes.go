package main

import (
	"net/http"
	"time"

	"github.com/Mirnda/mirandaclin/internal/cep"
	"github.com/Mirnda/mirandaclin/internal/domain/appointment"
	"github.com/Mirnda/mirandaclin/internal/domain/clinic"
	"github.com/Mirnda/mirandaclin/internal/domain/collaborator"
	"github.com/Mirnda/mirandaclin/internal/domain/consultation"
	"github.com/Mirnda/mirandaclin/internal/domain/invite"
	"github.com/Mirnda/mirandaclin/internal/domain/profile"
	"github.com/Mirnda/mirandaclin/internal/domain/user"
	"github.com/Mirnda/mirandaclin/internal/health"
	"github.com/Mirnda/mirandaclin/internal/infra/cache"
	"github.com/Mirnda/mirandaclin/internal/middleware"
	"github.com/Mirnda/mirandaclin/pkg/config"
	"github.com/Mirnda/mirandaclin/pkg/logger"
	"github.com/Mirnda/mirandaclin/pkg/response"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	_ "github.com/Mirnda/mirandaclin/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

type handlers struct {
	user         *user.Handler
	invite       *invite.Handler
	profile      *profile.Handler
	collaborator *collaborator.Handler
	clinic       *clinic.Handler
	appointment  *appointment.Handler
	consultation *consultation.Handler
	health       *health.Handler
	cep          *cep.Handler
}

// registerRoutes registra todas as rotas no mux e retorna o handler com o stack global de middlewares aplicado.
func registerRoutes(mux *http.ServeMux, h handlers, cfg *config.Config, c cache.Cache, log logger.Logger) http.Handler {
	publicRL := middleware.RateLimit(c, 10, time.Minute)

	authMw := middleware.Auth(cfg.JWT.Secret)

	generalRL := middleware.RateLimit(c, 120, time.Minute)
	protect := func(handler http.Handler) http.Handler {
		return generalRL(authMw(handler))
	}

	inviteRL := middleware.RateLimit(c, 5, time.Minute)
	inviteProtect := func(handler http.Handler) http.Handler {
		return inviteRL(authMw(handler))
	}

	reportRL := middleware.RateLimit(c, 30, time.Minute)
	reportProtect := func(handler http.Handler) http.Handler {
		return reportRL(authMw(handler))
	}

	// Swagger — disponível apenas fora de produção
	if cfg.App.Env != "production" {
		mux.Handle("GET /swagger/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
		))

		mux.Handle("GET /swagger/health/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"health"`,
			}),
		))
		mux.Handle("GET /swagger/auth/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"auth"`,
			}),
		))
		mux.Handle("GET /swagger/users/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"users"`,
			}),
		))
		mux.Handle("GET /swagger/profiles/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"profiles"`,
			}),
		))
		mux.Handle("GET /swagger/collaborators/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"collaborators"`,
			}),
		))
		mux.Handle("GET /swagger/clinics/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"clinics"`,
			}),
		))
		mux.Handle("GET /swagger/invites/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"invites"`,
			}),
		))
		mux.Handle("GET /swagger/appointments/", httpSwagger.Handler(
			httpSwagger.URL("/swagger/doc.json"),
			httpSwagger.UIConfig(map[string]string{
				"filter": `"appointments"`,
			}),
		))
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, "OK", map[string]any{
			"message": "Server running",
		})
	})

	// Observabilidade — sem autenticação (proteger por Security Group na AWS)
	mux.Handle("GET /metrics", promhttp.Handler())
	mux.HandleFunc("GET /health", h.health.Liveness)
	mux.HandleFunc("GET /health/ready", h.health.Readiness)

	// Auth — rotas públicas
	mux.Handle("POST /v1/api/auth/register", publicRL(http.HandlerFunc(h.user.Register)))
	mux.Handle("GET /v1/api/auth/verify-email", publicRL(http.HandlerFunc(h.user.VerifyEmail)))
	mux.Handle("POST /v1/api/auth/login", publicRL(http.HandlerFunc(h.user.Login)))
	mux.Handle("POST /v1/api/auth/refresh", publicRL(http.HandlerFunc(h.user.Refresh)))

	// Invites
	mux.Handle("POST /v1/api/invites", inviteProtect(http.HandlerFunc(h.invite.Create)))
	mux.Handle("POST /v1/api/invites/accept", publicRL(http.HandlerFunc(h.user.AcceptInvite)))

	//Users
	mux.Handle("GET /v1/api/users", protect(http.HandlerFunc(h.user.List)))
	mux.Handle("GET /v1/api/users/{id}", protect(http.HandlerFunc(h.user.GetByID)))
	mux.Handle("PUT /v1/api/users/{id}", protect(http.HandlerFunc(h.user.Update)))
	mux.Handle("DELETE /v1/api/users/{id}", protect(http.HandlerFunc(h.user.Delete)))

	//Collaborators
	mux.Handle("POST /v1/api/collaborators", protect(http.HandlerFunc(h.collaborator.Create)))
	mux.Handle("GET /v1/api/collaborators", protect(http.HandlerFunc(h.collaborator.List)))

	//Profiles
	mux.Handle("POST /v1/api/profiles", protect(http.HandlerFunc(h.profile.Create)))
	mux.Handle("GET /v1/api/profiles/role/{role}", protect(http.HandlerFunc(h.profile.ListByRole)))
	mux.Handle("GET /v1/api/profiles/{id}", protect(http.HandlerFunc(h.profile.GetByID)))
	mux.Handle("PUT /v1/api/profiles/{id}", protect(http.HandlerFunc(h.profile.Update)))
	mux.Handle("DELETE /v1/api/profiles/{id}", protect(http.HandlerFunc(h.profile.Delete)))

	// Clinics
	mux.Handle("POST /v1/api/clinics", protect(http.HandlerFunc(h.clinic.Create)))
	mux.Handle("GET /v1/api/clinics", protect(http.HandlerFunc(h.clinic.List)))
	mux.Handle("GET /v1/api/clinics/{id}", protect(http.HandlerFunc(h.clinic.GetByID)))
	mux.Handle("PUT /v1/api/clinics/{id}", protect(http.HandlerFunc(h.clinic.Update)))
	mux.Handle("DELETE /v1/api/clinics/{id}", protect(http.HandlerFunc(h.clinic.Delete)))

	// Appointments
	mux.Handle("POST /v1/api/appointments", protect(http.HandlerFunc(h.appointment.Create)))
	mux.Handle("GET /v1/api/appointments/patient/{patient_id}", protect(http.HandlerFunc(h.appointment.ListByPatient)))
	mux.Handle("PATCH /v1/api/appointments/{id}/cancel", protect(http.HandlerFunc(h.appointment.Cancel)))

	// CEP — consulta endereço por CEP em APIs públicas
	mux.Handle("GET /v1/api/cep/{cep}", protect(http.HandlerFunc(h.cep.Lookup)))

	// Consultations — rate limit reduzido por ser rota de relatório
	mux.Handle("POST /v1/api/consultations", protect(http.HandlerFunc(h.consultation.Create)))
	mux.Handle("GET /v1/api/consultations/patient/{patient_id}", reportProtect(http.HandlerFunc(h.consultation.ListByPatient)))
	mux.Handle("GET /v1/api/consultations/dentist/{dentist_id}", reportProtect(http.HandlerFunc(h.consultation.ListByDentist)))

	// Stack global: RequestID → RequestLogger → SecurityHeaders → CORS → Metrics → rotas
	return middleware.RequestID(
		middleware.RequestLogger(log)(
			middleware.SecurityHeaders(cfg.App.Env)(
				middleware.CORS(cfg.App.CORSAllowedOrigins)(
					middleware.Metrics(mux),
				),
			),
		),
	)
}
