package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net/http"
	"strconv"
	"time"

	"github.com/Lit3D/lit3d-kuka-c3-gate/app"
)

const (
  Service_StartEndTimeout = 1 * time.Second
  Service_Bots_API = "/bots"
)

type Service struct {
  botsTeam *Team
  mux      *http.ServeMux
  server   *http.Server
}

func init() {
  mime.AddExtensionType(".js"  , "application/javascript")
  mime.AddExtensionType(".mjs" , "application/javascript")
  
  mime.AddExtensionType(".css" , "text/css")
  mime.AddExtensionType(".tpl" , "text/html")
  mime.AddExtensionType(".html", "text/html")
}

func NewService(port PortValue, botsTeam *Team) *Service {
  service := &Service{
    botsTeam: botsTeam,
    mux: http.NewServeMux(),
  }

  service.mux.Handle("/", http.FileServer(app.AppFS))
  service.mux.HandleFunc(Service_Bots_API, service.BotHandler)

  service.server = &http.Server{
    Addr:    fmt.Sprintf(":%s", port.String()),
    Handler: service.mux,
  }

  return service
}

func (service *Service) ListenAndServe() error {
  errChan := make(chan error, 1)

  go func() {
    log.Printf("[Service INFO] Listening on http://0.0.0.0%s\n", service.server.Addr)
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

    case <-time.After(Service_StartEndTimeout):
      return nil
  }

  return nil
}

func (service *Service) Shutdown() error {
  ctx, cancel := context.WithTimeout(context.Background(), Service_StartEndTimeout)
  defer cancel()
  
  if err := service.server.Shutdown(ctx); err != nil {
    return err
  }

  log.Printf("[Service INFO] App stopped\n")
  return nil
}

func (service *Service) BotHandler(w http.ResponseWriter, r *http.Request) {
  switch r.Method {
    case "GET":
      teamAppData := service.botsTeam.GetAppData()
      w.WriteHeader(http.StatusOK)
      w.Header().Set("Content-Type", "application/json; charset=utf-8")
      if err := json.NewEncoder(w).Encode(teamAppData); err != nil {
        log.Printf("[Service ERROR] Get parse json error: %v\n", err)
      }
      return
    case "POST":
      r.ParseMultipartForm(10 << 25)
      botIdStr := r.FormValue("botId")
      moveGroupIdStr := r.FormValue("moveGroupId")

      botId, err := strconv.ParseUint(botIdStr, 10, 16)
      if err != nil {
        log.Printf("[Service ERROR] POST parse Bot ID error: %v\n", err)
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
      }

      moveGroupId, err := strconv.ParseUint(moveGroupIdStr, 10, 16)
      if err != nil {
        log.Printf("[Service ERROR] POST parse MoveGroup ID error: %v\n", err)
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
      }
      
      bot := service.botsTeam.Bots[int(botId)]
      if bot == nil {
        log.Printf("[Service ERROR] POST Bot is not found of id %d\n", botId)
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
      }

      if err := bot.RunMoveGroup(uint16(moveGroupId)); err != nil {
        log.Printf("[Service ERROR] POST Run MoveGroup error: %v\n", err)
        http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
        return
      }

      w.WriteHeader(http.StatusOK)
      w.Header().Set("Content-Type", "application/json; charset=utf-8")
      var trueData bool = true
      json.NewEncoder(w).Encode(trueData)
      return
    
    default:
      http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
      return
    }
}
