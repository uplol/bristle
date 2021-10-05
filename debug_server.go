package bristle

import (
	"context"
	"net/http"
	_ "net/http/pprof"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

type debugServer struct {
	closer chan struct{}
	config DebuggingConfig
	http   *http.Server
}

func newDebugServer(config DebuggingConfig) *debugServer {
	http := http.Server{
		Addr: config.Bind,
	}
	return &debugServer{closer: make(chan struct{}), config: config, http: &http}
}

func (d *debugServer) Close() {
	d.closer <- struct{}{}
	<-d.closer
}

func (d *debugServer) Run() {
	if d.config.BlockProfileRate != nil {
		runtime.SetBlockProfileRate(*d.config.BlockProfileRate)
	} else {
		runtime.SetBlockProfileRate(0)
	}

	if d.config.MutexProfileFraction != nil {
		runtime.SetMutexProfileFraction(*d.config.MutexProfileFraction)
	} else {
		runtime.SetMutexProfileFraction(0)
	}

	if d.config.Metrics {
		http.Handle("/metrics", promhttp.Handler())
	}

	errChan := make(chan error)
	go func() {
		log.Info().Msg("debug-server: starting http server")
		err := d.http.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Error().Err(err).Msg("debug-server: http server error")
			errChan <- err
		}
		close(errChan)
	}()

	select {
	case err := <-errChan:
		log.Error().Err(err).Msg("debug-server: http listener exited")
	case <-d.closer:
		log.Info().Msg("debug-server: shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		d.http.Shutdown(shutdownCtx)
		cancel()
		<-errChan
		log.Info().Msg("debug-server: shutdown completed")
		close(d.closer)
	}
}
