// @title          mirandaclin API
// @version        1.0
// @description    SaaS multi-tenant para clínicas odontológicas
// @host           localhost:8080
// @BasePath       /
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
package main

import (
	"fmt"
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
	infraCache "github.com/Mirnda/mirandaclin/internal/infra/cache"
	infraDB "github.com/Mirnda/mirandaclin/internal/infra/db"
	"github.com/Mirnda/mirandaclin/internal/infra/repository"
	"github.com/Mirnda/mirandaclin/pkg/config"
	"github.com/Mirnda/mirandaclin/pkg/logger"
	"github.com/Mirnda/mirandaclin/pkg/mailer"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // silencioso — em ECS as vars vêm do ambiente

	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("config inválida: %v", err))
	}

	log := logger.New(cfg.App.Env)
	defer func() { _ = log.Sync() }()

	// Banco de dados
	db, err := infraDB.New(cfg.DB.DSN())
	if err != nil {
		log.Error("falha ao conectar banco", logger.Err(err))
		panic(err)
	}
	if err := infraDB.Migrate(db); err != nil {
		log.Error("falha na migration", logger.Err(err))
		panic(err)
	}

	// Cache
	cache := infraCache.New(log, cfg.Redis)

	// Repositórios
	userRepo := repository.NewUserRepository()
	profileRepo := repository.NewProfileRepository()
	inviteRepo := repository.NewInviteRepository()
	tenantRepo := repository.NewTenantRepository()
	clinicRepo := repository.NewClinicRepository()
	pcRepo := repository.NewProfileClinicRepository()
	blockRepo := repository.NewProfileBlockRepository()
	apptRepo := repository.NewAppointmentRepository()
	consultRepo := repository.NewConsultationRepository()

	// Mailer
	ml := mailer.New(cfg.Mailer)

	// Services
	userSvc := user.NewService(db, userRepo, profileRepo, inviteRepo, tenantRepo, cfg.App, ml, cfg.JWT.Secret)
	profileSvc := profile.NewService(db, cache, profileRepo)
	collaboratorSvc := collaborator.NewService(db, cache, profileRepo, clinicRepo, pcRepo, blockRepo)
	inviteSvc := invite.NewService(db, inviteRepo, profileRepo, ml, cfg.App)
	clinicSvc := clinic.NewService(db, clinicRepo)
	apptSvc := appointment.NewService(db, apptRepo, pcRepo, blockRepo)
	consultSvc := consultation.NewService(db, consultRepo)

	// Handlers
	h := handlers{
		user:         user.NewHandler(userSvc, cfg.App.Env),
		invite:       invite.NewHandler(inviteSvc),
		profile:      profile.NewHandler(profileSvc),
		collaborator: collaborator.NewHandler(collaboratorSvc),
		clinic:       clinic.NewHandler(clinicSvc),
		appointment:  appointment.NewHandler(apptSvc),
		consultation: consultation.NewHandler(consultSvc),
		cep:          cep.NewHandler(cache),
		health:       health.NewHandler(db, cache),
	}

	mux := http.NewServeMux()
	handler := registerRoutes(mux, h, cfg, cache, log)

	addr := ":" + cfg.App.Port
	log.Info("servidor iniciando", logger.String("addr", addr))

	srv := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := srv.ListenAndServe(); err != nil {
		log.Error("servidor encerrado", logger.Err(err))
	}
}
