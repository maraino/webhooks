package server

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/maraino/webhooks/pkg/types"
	"github.com/smallstep/logging/httplog"
)

// Webhook is an http handler that shows if a device is registered in the
// database.
type Webhook struct {
	DB *sql.DB
}

// ServeHTTP implements the http.Handler interface on the Webhook type.
func (srv *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.RequestURI {
	case "/devices":
		srv.devices(w, r)
	default:
		httpError(w, http.StatusNotFound, nil)
	}
}

func (srv *Webhook) devices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		httpError(w, http.StatusMethodNotAllowed, nil)
		return
	}

	var body types.RequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		httpError(w, http.StatusBadRequest, err)
		return
	}

	if body.AttestationData == nil || body.AttestationData.PermanentIdentifier == "" {
		httpError(w, http.StatusBadRequest, nil)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	device, err := types.LoadDevice(ctx, srv.DB, body.AttestationData.PermanentIdentifier)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			httpError(w, http.StatusInternalServerError, err)
			return
		}
		device = &types.Device{}
	}

	resp := &types.ResponseBody{
		Allow: device.Allow,
		Data:  json.RawMessage(device.Data),
	}

	logMessage(w, "allow device %s: %v", device.ID, device.Allow)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		httpError(w, http.StatusInternalServerError, err)
		return
	}
}

func httpError(w http.ResponseWriter, status int, err error) {
	if err != nil {
		logError(w, err)
	}
	http.Error(w, http.StatusText(status), status)
}

func logError(w http.ResponseWriter, err error) {
	if rl, ok := w.(httplog.ResponseLogger); ok {
		rl.WithField(httplog.ErrorKey, err)
	}
}

func logMessage(w http.ResponseWriter, format string, args ...any) {
	if rl, ok := w.(httplog.ResponseLogger); ok {
		rl.WithField(httplog.MessageKey, fmt.Sprintf(format, args...))
	}
}
