package main

import (
  "context"
  "fmt"
  "log"
  "mime"
  "net/http"
  "time"

  "github.com/Lit3D/lit3d-kuka-c3-gate/app"
)

const (
  App_StartEndTimeout = 1 * time.Second
)

type Service struct {
  mux    *http.ServeMux
  server *http.Server
}

func init() {
  mime.AddExtensionType(".js", "application/javascript")
  mime.AddExtensionType(".mjs", "application/javascript")
  mime.AddExtensionType(".css", "text/css")
  mime.AddExtensionType(".tpl", "text/html")
  mime.AddExtensionType(".html", "text/html")
}

func NewService(port PortValue) *Service {
  service := &Service{
    mux: http.NewServeMux(),
  }

  service.mux.Handle("/", http.FileServer(app.AppFS))
  service.mux.HandleFunc("/bot", service.BotHandler)

  service.server = &http.Server{
    Addr:    fmt.Sprintf(":%s", port.String()),
    Handler: service.mux,
  }

  return service
}

func (service *Service) ListenAndServe() error {
  errChan := make(chan error, 1)

  go func() {
    log.Printf("[APP INFO] Listening on http://0.0.0.0%s\n", service.server.Addr)
    if err := service.server.ListenAndServe(); err != http.ErrServerClosed {
      errChan <- err
    }
    close(errChan)
  }()

  select {
    case err := <-errChan:
      if err != nil {
        return err
      }

    case <-time.After(App_StartEndTimeout):
      return nil
  }

  return nil
}

func (service *Service) Shutdown() error {
  ctx, cancel := context.WithTimeout(context.Background(), App_StartEndTimeout)
  defer cancel()
  
  if err := service.server.Shutdown(ctx); err != nil {
    return err
  }

  log.Printf("[APP INFO] App stopped\n")
  return nil
}

func (service *Service) BotHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case "GET":
      fmt.Fprintf(w, "Handling GET request")
    case "POST":
      fmt.Fprintf(w, "Handling POST request")
    default:
      http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}