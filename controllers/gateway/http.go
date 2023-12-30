package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Gateway struct {
	client.Client
	Recorder record.EventRecorder
}

var _ manager.Runnable = &Gateway{}

func (g *Gateway) Start(ctx context.Context) error {
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.RealIP)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(middleware.Recoverer)

	r.Get("/problem/{name}", g.GetProblemEnvironmentController)
	r.Post("/problem", g.AcquireProblemEnvironmentHandler)
	r.Delete("/problem/{name}", g.ReleaseProblemEnvironmentHandler)

	server := http.Server{
		Addr: ":8082",
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: r,
	}

	// serverErrChan forwards errors from HTTP server error from HTTP goroutine.
	serverErrChan := make(chan error, 1)
	go func() {
		// ListenAndServe will return error in the following reasons:
		//  * Server is closed by Shutdown
		//  * Underlying Listener can't be worked by some reasons
		// The first error can be ignored, but the second one should be handled correctly.
		// To return the error to the caller of Start(), that error will forward to serverErrChan.
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			serverErrChan <- err
		}
	}()

	select {
	case err := <-serverErrChan:
		// in the case that ListenAndServer returns error.
		// In this case, we don't have to call Shutdown as the server is not started actually.
		return err
	case <-ctx.Done():
		// in the case that Shutdown is called.
		// In this case, we need to call Shutdown to gracefully shutdown the server.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		return server.Shutdown(ctx)
	}
}

func (g *Gateway) GetProblemEnvironmentController(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	problemEnvironment, err := g.GetProblemEnvironment(ctx, name)
	if err != nil {
		if _, ok := AsErrProblemEnvironmentNotFound(err); ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if _, ok := AsErrWorkerNotFound(err); ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("worker not found"))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	response := GetProblemEnvironmentResponse(*problemEnvironment)
	if err := renderJSON(w, response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) AcquireProblemEnvironmentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	request := AcquireProblemEnvironmentRequest{}
	if err := bindJSON(r, &request); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	problemEnvironment, err := g.AcquireProblemEnvironment(ctx, request.ProblemName)
	if err != nil {
		if _, ok := AsErrProblemNotFound(err); ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if _, ok := AsErrNoAvailableProblemEnvironment(err); ok {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		if _, ok := AsErrWorkerNotFound(err); ok {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("worker not found"))
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	response := AcquireProblemEnvironmentResponse(*problemEnvironment)
	if err := renderJSON(w, response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func (g *Gateway) ReleaseProblemEnvironmentHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	name := chi.URLParam(r, "name")

	if err := g.ReleaseProblemEnvironment(ctx, name); err != nil {
		if _, ok := AsErrProblemEnvironmentNotFound(err); ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusOK)
}

func bindJSON[T any](r *http.Request, v T) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(v); err != nil {
		return err
	}
	return nil
}

func renderJSON[T any](w http.ResponseWriter, response T) error {
	buf := bytes.Buffer{}

	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(true)
	if err := encoder.Encode(response); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(buf.Bytes())
	return nil
}
