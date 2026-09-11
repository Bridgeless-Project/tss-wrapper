package api

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/config"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/core/scheduler"

	"github.com/Bridgeless-Project/tss-wrapper-svc/docs"
	srvgrpc "github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc"
	"github.com/Bridgeless-Project/tss-wrapper-svc/internal/api/grpc/ctx"
	db "github.com/Bridgeless-Project/tss-wrapper-svc/internal/data"
	types "github.com/Bridgeless-Project/tss-wrapper-svc/resources"
	"github.com/go-chi/chi/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"gitlab.com/distributed_lab/ape"
	"gitlab.com/distributed_lab/logan/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/ignite/cli/ignite/pkg/openapiconsole"
)

type Server struct {
	grpc net.Listener
	http net.Listener

	logger       *logan.Entry
	ctxExtenders []func(context.Context) context.Context
}

// NewServer creates a new GRPC server.
func NewServer(
	grpc net.Listener,
	http net.Listener,
	tasksDB db.TasksQ,
	logger *logan.Entry,
	scheduler *scheduler.Scheduler,
	tssConfig *config.TSSConfig,
) *Server {
	return &Server{
		grpc:   grpc,
		http:   http,
		logger: logger,

		ctxExtenders: []func(context.Context) context.Context{
			ctx.LoggerProvider(logger),
			ctx.DBProvider(tasksDB),
			ctx.SchedulerProvider(scheduler),
			ctx.TSSConfigProvider(tssConfig),
		},
	}
}

func (s *Server) RunGRPC(ctx context.Context) error {
	srv := s.grpcServer()

	// graceful shutdown
	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		s.logger.Debug("grpc serving stopped: context canceled")
	}()

	s.logger.Debug("grpc serving started")
	return srv.Serve(s.grpc)
}

func (s *Server) RunHTTP(ctxt context.Context) error {
	handler, err := s.httpRouter(ctxt)
	if err != nil {
		return err
	}
	srv := &http.Server{Handler: handler}

	// graceful shutdown
	go func() {
		<-ctxt.Done()
		shutdownDeadline, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownDeadline); err != nil {
			s.logger.WithError(err).Error("failed to shutdown http server")
		}
		s.logger.Debug("http serving stopped: context canceled")
	}()

	s.logger.Debug("http serving started")
	if err := srv.Serve(s.http); !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

func (s *Server) httpRouter(ctxt context.Context) (http.Handler, error) {
	router := chi.NewRouter()
	router.Use(
		ape.LoganMiddleware(s.logger),
		ape.RecoverMiddleware(s.logger),
		ape.CtxMiddleware(s.ctxExtenders...),
	)

	// pointing to grpc implementation
	grpcGatewayRouter := runtime.NewServeMux()
	err := types.RegisterAPIHandlerServer(ctxt, grpcGatewayRouter, srvgrpc.Implementation{})
	if err != nil {

		return nil, errors.Wrap(err, "failed to register api handler")
	}

	router.Mount("/", grpcGatewayRouter)
	router.Handle("/static/*", http.FileServer(http.FS(docs.Docs)))
	router.HandleFunc("/api", openapiconsole.Handler("TSS wrapper service API", "/static/api.swagger.json"))

	return router, nil
}

func (s *Server) grpcServer() *grpc.Server {
	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(),
	)

	types.RegisterAPIServer(srv, srvgrpc.Implementation{})
	reflection.Register(srv)

	return srv
}
