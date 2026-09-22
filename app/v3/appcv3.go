package v3

import (
	pkgCfgV3 "axgit.vixiv.ru/snake/arcella-lib/app/v3/config/basic"
	"context"
	"log/slog"
	"os"
	"os/signal"
	"reflect"
	"syscall"
	"time"
)

var (
	compilationDate = ""
)

type Service interface {
	GetName() string
	Start(ctx context.Context) error
	Stop() error
}

type Application interface {
	Start() (services []Service)
	Stop() (services []Service)
}

type ConfigManager interface {
	Load() error
}

type AppControllerV3 struct {
	app    Application
	logger *slog.Logger
	start  time.Time
	ctx    context.Context
	quit   chan os.Signal
	//
	cm ConfigManager
	// timer func
	processingDuration time.Duration
	processingFunc     func()
	timerFunk          func()
	fooChan            chan byte
}

func NewAppControllerV3(app Application) *AppControllerV3 {
	return &AppControllerV3{start: time.Now(), app: app, fooChan: make(chan byte, 1)}
}

func (v3 *AppControllerV3) SetLogHandler(handler slog.Handler) *AppControllerV3 {
	v3.logger = slog.New(handler)
	slog.SetDefault(v3.logger)
	slog.Debug("SetLogHandler")
	return v3
}

func (v3 *AppControllerV3) GetLogHandler() *slog.Logger {
	slog.Debug("GetLogHandler", slog.Any("logger", v3.logger))
	return v3.logger
}

func (v3 *AppControllerV3) SetConfigManager(configManager ConfigManager) *AppControllerV3 {
	v3.cm = configManager
	return v3
}

// TODO: !!!
func (v3 *AppControllerV3) checkConfigManager() {
	if v3.cm == nil {
		slog.Debug("checkConfigManager")
		r := reflect.ValueOf(v3.app)
		confField := reflect.Indirect(r).FieldByName("Config")
		v3.cm = pkgCfgV3.NewBasicConfig(confField.Interface())
	}
}

func (v3 *AppControllerV3) Run() *AppControllerV3 {
	// Recover function
	if thisApp, ok := v3.app.(interface{ OnAppRecover(err any) }); ok {
		slog.Info("OnAppRecover")
		defer func() {
			slog.Info("OnAppRecover event")
			if r := recover(); r != nil {
				slog.Info("OnAppRecover action")
				thisApp.OnAppRecover(r)
			}
		}()
	}

	// TODO: DO NOT SIMPLIFY!!!
	if compilationDate != "" { // Do not simplify!!!
		slog.Info("Current", slog.String("mark", compilationDate))
	}

	// v3.checkConfigManager()
	var err error

	// config section
	if v3.cm == nil {
		slog.Debug("No config manager")
	} else {
		// slog.Debug("Ready to load config")
		err = v3.cm.Load()
		slog.Debug("Config loaded")
	}

	// Init
	if err == nil {

		// Create context //
		ctx, cancel := context.WithCancel(context.Background())
		v3.ctx = ctx

		// Function before init services
		if thisApp, ok := v3.app.(interface{ OnAppBeforeInitServices() }); ok {
			slog.Debug("OnAppBeforeInitServices")
			thisApp.OnAppBeforeInitServices()
		}

		// Init services
		v3.initServices(ctx)

		// Function after init services
		if thisApp, ok := v3.app.(interface{ OnAppAfterInitServices() }); ok {
			slog.Debug("OnAppAfterInitServices")
			thisApp.OnAppAfterInitServices()
		}

		// Timer function
		if thisApp, ok := v3.app.(interface {
			GetProcessingFunc() (func(), time.Duration)
		}); ok {
			v3.processingFunc, v3.processingDuration = thisApp.GetProcessingFunc()
			if v3.processingDuration > 0 {
				v3.timerFunk = func() { time.Sleep(v3.processingDuration); v3.fooChan <- 0 }
			}
			slog.Debug("Processing timer", slog.Duration("duration", v3.processingDuration))
		}

		// Starting
		v3.quit = make(chan os.Signal, 1)
		signal.Notify(v3.quit, os.Interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		defer signal.Stop(v3.quit)
		v3.doProcess() // invoke main procedure

		// Stopping
		v3.deInitServices()
		cancel()

		slog.Debug("Work", slog.Duration("time", time.Since(v3.start)))

	} else {
		panic(err)
	}

	return v3
}

func (v3 *AppControllerV3) Stop() {
	slog.Info("Stop")
	select {
	case v3.quit <- os.Interrupt:
	default:
	}
}

func (v3 *AppControllerV3) initServices(ctx context.Context) {
	slog.Info("initServices")
	var started []Service
	for k, v := range v3.app.Start() {
		serviceName := v.GetName()
		slog.Info("Service", slog.Int("id", k), slog.String("name", serviceName))
		if err := v.Start(ctx); err != nil {
			slog.Error("Error in service", "name", serviceName, "error", err)
			for i := len(started) - 1; i >= 0; i-- {
				if serr := started[i].Stop(); serr != nil {
					slog.Warn("Error stopping service", "name", started[i].GetName(), "error", serr)
				}
			}
			panic(err)
		}
		started = append(started, v)
	}
}

func (v3 *AppControllerV3) deInitServices() {
	slog.Info("deInitServices")
	var err error
	var serviceName string
	for n, service := range v3.app.Stop() {
		if service != nil {
			serviceName = service.GetName()
			slog.Info("Service", slog.Int("num", n), slog.String("name", serviceName))
			if err = service.Stop(); err != nil {
				slog.Warn("Error in service", "name", serviceName, "error", err)
			}
		} else {
			slog.Warn("Service is nil")
		}
	}
}

func (v3 *AppControllerV3) doProcess() {
	slog.Info("Starting", slog.Duration("time", time.Since(v3.start)))
	if v3.processingFunc != nil && v3.processingDuration > 0 {
		ticker := time.NewTicker(v3.processingDuration)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				v3.processingFunc()
			case <-v3.quit:
				slog.Debug("Quit")
				return
			}
		}
	} else {
		for {
			select {
			case <-v3.quit:
				slog.Debug("Quit")
				return
			}
		}
	}
}

func (v3 *AppControllerV3) GetContext() context.Context {
	return v3.ctx
}
