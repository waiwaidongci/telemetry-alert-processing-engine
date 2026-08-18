package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/example/telemetry-alert/api"
	"github.com/example/telemetry-alert/api/router"
	"github.com/example/telemetry-alert/internal/adapter/metrics"
	"github.com/example/telemetry-alert/internal/adapter/notifier"
	"github.com/example/telemetry-alert/internal/adapter/ratelimit"
	"github.com/example/telemetry-alert/internal/application/alertrule"
	deviceapp "github.com/example/telemetry-alert/internal/application/device"
	eventapp "github.com/example/telemetry-alert/internal/application/event"
	notificationapp "github.com/example/telemetry-alert/internal/application/notification"
	retentionapp "github.com/example/telemetry-alert/internal/application/retention"
	telemetryapp "github.com/example/telemetry-alert/internal/application/telemetry"
	tenantapp "github.com/example/telemetry-alert/internal/application/tenant"
	"github.com/example/telemetry-alert/internal/config"
	alertrepo "github.com/example/telemetry-alert/internal/infrastructure/repository/alert"
	devicerepo "github.com/example/telemetry-alert/internal/infrastructure/repository/device"
	notificationrepo "github.com/example/telemetry-alert/internal/infrastructure/repository/notification"
	retentionrepo "github.com/example/telemetry-alert/internal/infrastructure/repository/retention"
	telemetryrepo "github.com/example/telemetry-alert/internal/infrastructure/repository/telemetry"
	tenantrepo "github.com/example/telemetry-alert/internal/infrastructure/repository/tenant"
	"github.com/example/telemetry-alert/internal/infrastructure/sqlite"
	"github.com/example/telemetry-alert/internal/system"
)

type App struct {
	cfg        config.Config
	logger     *slog.Logger
	db         *sql.DB
	httpServer *http.Server
	metrics    *api.Metrics
	evaluator  *alertrule.Evaluator
	dispatcher *notificationapp.Dispatcher
	retention  *retentionapp.Service
	ready      atomic.Bool

	bgCancel context.CancelFunc
	bgWg     sync.WaitGroup
}

func New(cfg config.Config, logger *slog.Logger) (*App, error) {
	if cfg.Database.Driver != "sqlite" {
		return nil, fmt.Errorf("unsupported development database driver %q; this build supports sqlite", cfg.Database.Driver)
	}
	db, err := sqlite.Open(cfg.Database.SQLitePath)
	if err != nil {
		return nil, err
	}
	if err := sqlite.Migrate(context.Background(), db); err != nil {
		db.Close()
		return nil, err
	}
	db.SetMaxOpenConns(cfg.Database.MaxOpenConns)
	db.SetMaxIdleConns(cfg.Database.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.Database.ConnMaxLifetime)

	ids := system.UUIDGenerator{}
	clock := system.RealClock{}
	tokens := system.SHA256TokenHasher{}

	tenantRepo := tenantrepo.NewRepository(db)
	deviceRepo := devicerepo.NewRepository(db)
	telemetryRepo := telemetryrepo.NewRepository(db)
	ruleRepo := alertrepo.NewRuleRepository(db)
	stateRepo := alertrepo.NewStateRepository(db)
	eventRepo := alertrepo.NewEventRepository(db)
	notificationRepo := notificationrepo.NewRepository(db)
	retentionCleaner := retentionrepo.NewCleaner(db)

	tenantService := tenantapp.NewService(tenantRepo, ids, clock)
	deviceService := deviceapp.NewService(deviceRepo, ids, clock, tokens)
	eventService := eventapp.NewService(eventRepo)

	limit := ratelimit.NewTokenBucketLimiter(cfg.Telemetry.IngestRateLimit, cfg.Telemetry.IngestBurst)
	windowReader := metrics.NewWindowReader(telemetryRepo)
	sender := notifier.NewSender(cfg.Notification.WebhookURL, cfg.Notification.RequestTimeout, logger)
	dispatcher := notificationapp.NewDispatcher(notificationRepo, sender, eventService, ids, clock, logger, notificationapp.Config{
		MaxAttempts:    cfg.Notification.MaxAttempts,
		InitialBackoff: cfg.Notification.InitialBackoff,
		MaxBackoff:     cfg.Notification.MaxBackoff,
		RequestTimeout: cfg.Notification.RequestTimeout,
		WorkerCount:    cfg.Notification.WorkerCount,
	})
	evaluator := alertrule.NewEvaluator(ruleRepo, deviceService, stateRepo, eventService, windowReader, dispatcher, clock, ids, logger, cfg.Notification.WorkerCount)

	telemetryService := telemetryapp.NewService(telemetryRepo, telemetryRepo, evaluator, limit, ids, clock, telemetryapp.Config{
		MaxBatchSize:      cfg.Telemetry.MaxBatchSize,
		MaxPointAge:       cfg.Telemetry.MaxPointAge,
		MaxFutureSkew:     cfg.Telemetry.MaxFutureSkew,
		QueryDefaultLimit: cfg.Telemetry.QueryDefaultLimit,
		QueryMaxLimit:     cfg.Telemetry.QueryMaxLimit,
	})
	ruleService := alertrule.NewService(ruleRepo, stateRepo, ids, clock)
	retentionService := retentionapp.NewService(retentionCleaner, clock, retentionapp.Config{
		RawRetention:       cfg.Retention.RawRetention,
		AggregateRetention: cfg.Retention.AggregateRetention,
		EventRetention:     cfg.Retention.EventRetention,
		FailureRetention:   cfg.Retention.FailureRetention,
		DeleteBatchSize:    cfg.Retention.DeleteBatchSize,
	})

	app := &App{
		cfg:        cfg,
		logger:     logger,
		db:         db,
		metrics:    api.NewMetrics(),
		evaluator:  evaluator,
		dispatcher: dispatcher,
		retention:  retentionService,
	}
	app.httpServer = &http.Server{
		Addr: cfg.Server.Addr,
		Handler: router.New(router.Dependencies{
			DB:            db,
			Logger:        logger,
			Metrics:       app.metrics,
			Ready:         app.ready.Load,
			Tenants:       tenantService,
			Devices:       deviceService,
			Telemetry:     telemetryService,
			Rules:         ruleService,
			Events:        eventService,
			Notifications: dispatcher,
		}),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	bgCtx, cancel := context.WithCancel(ctx)
	a.bgCancel = cancel
	a.evaluator.Start()
	a.dispatcher.Start()
	a.ready.Store(true)
	a.startBackground(bgCtx, a.cfg.Alert.EvaluationInterval, func(c context.Context) {
		if err := a.evaluator.Sweep(c); err != nil {
			a.logger.Error("alert sweep failed", "error", err)
		}
	})
	a.startBackground(bgCtx, time.Second, func(c context.Context) {
		if err := a.dispatcher.RetryPending(c); err != nil {
			a.logger.Error("notification retry failed", "error", err)
		}
	})
	a.startBackground(bgCtx, a.cfg.Retention.RunInterval, func(c context.Context) {
		if _, err := a.retention.Run(c); err != nil {
			a.logger.Error("retention cleanup failed", "error", err)
		}
	})

	errCh := make(chan error, 1)
	go func() {
		a.logger.Info("http server starting", "addr", a.cfg.Server.Addr)
		if err := a.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		a.ready.Store(false)
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), a.cfg.Server.ShutdownTimeout)
		defer shutdownCancel()
		if err := a.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		cancel()
		a.evaluator.Stop()
		a.dispatcher.Stop()
		a.bgWg.Wait()
		if err := a.db.Close(); err != nil {
			return fmt.Errorf("close database: %w", err)
		}
		return nil
	}
}

func (a *App) startBackground(ctx context.Context, interval time.Duration, fn func(context.Context)) {
	a.bgWg.Add(1)
	go func() {
		defer a.bgWg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fn(ctx)
			case <-ctx.Done():
				return
			}
		}
	}()
}
