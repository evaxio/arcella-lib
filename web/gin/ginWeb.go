package gin

import (
	"axgit.vixiv.ru/snake/arcella-lib/utils"
	pkgF2B "axgit.vixiv.ru/snake/arcella-lib/web/gin/middleware/fail2ban"
	pkgRL "axgit.vixiv.ru/snake/arcella-lib/web/gin/middleware/requestLog"
	"errors"
	log "log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/context"
)

type GinWeb struct {
	config    *ConfigGinWeb
	engine    *gin.Engine
	srv       *http.Server
	startOnce sync.Once
}

// type GinOptions func(gw *GinWeb)

func NewGinWeb(config *ConfigGinWeb) *GinWeb { // , options ...GinOptions) *GinWeb {
	log.Info("GinWeb", log.Int("port", config.Port), log.String("StartMode", config.StartMode))
	startModeName := strings.ToLower(config.StartMode)
	if startModeName != gin.DebugMode && startModeName != gin.TestMode && startModeName != gin.ReleaseMode {
		log.Info("Set default debug mode")
		gin.SetMode(gin.DebugMode)
	} else {
		log.Info("Set start mode", log.String("mode", startModeName))
		gin.SetMode(startModeName)
	}
	gw := GinWeb{
		config: config,
		engine: gin.New(),
	}
	if err := gw.engine.SetTrustedProxies(nil); err != nil {
		log.Error("SetTrustedProxies", log.String("message", err.Error()))
	}
	gw.engine.Use(gin.Recovery())
	if config.UseLogger {
		gw.engine.Use(gin.Logger())
	}

	if config.RequestLog {
		gw.engine.Use(pkgRL.GetRequestLog())
	}

	if config.Fail2Ban.Disabled == false {
		gw.engine.Use(pkgF2B.GetFail2Ban(&config.Fail2Ban))
	}

	if config.Static != "" {
		log.Info("Static", log.String("dir", config.Static)) // TODO: check folder exists

		gw.engine.Static("/", config.Static)
		gw.engine.NoRoute(func(c *gin.Context) {
			c.Redirect(http.StatusFound, "/")
		})
	}
	// for _, option := range options {option(&gw)}
	return &gw
}

func (gw *GinWeb) GetName() string {
	return "GinWeb"
}

func (gw *GinWeb) Start(ctx context.Context) (err error) {
	gw.startOnce.Do(func() {
		gw.srv = &http.Server{
			Addr:              ":" + utils.IntToStr(gw.config.Port),
			Handler:           gw.engine,
			ErrorLog:          NewServerErrorLog(),
			ReadTimeout:       gw.config.ReadTimeOut, // 10 * time.Second,
			ReadHeaderTimeout: 10 * time.Second,
			WriteTimeout:      gw.config.WriteTimeOut, // 10 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		go func() {
			if err := gw.srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("ListenAndServe", log.String("Message", err.Error()))
			}
		}()
		go func() {
			<-ctx.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := gw.srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Error("Shutdown", log.String("Message", err.Error()))
			}
		}()
	})

	return err

}

func (gw *GinWeb) Stop() (err error) {
	if gw.srv == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	err = gw.srv.Shutdown(ctx)
	cancel()
	// defer cancel()
	//if err := g.srv.Shutdown(ctx); err != nil {log.Error("Server Shutdown:", err)}
	return err
}

func (gw *GinWeb) GetEngine() *gin.Engine {
	return gw.engine
}

// func OptionTLS(some string) GinOptions {return func(gw *GinWeb) {log.Debug("Set utils: ", some)}}
