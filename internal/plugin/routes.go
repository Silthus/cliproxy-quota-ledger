package plugin

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Silthus/cliproxy-quota-ledger/internal/ledger"
)

const (
	samplesRoute     = "/quota-ledger/samples"
	samplesPath      = "/v0/management" + samplesRoute
	maxPostedSamples = 2000
	maxServedSamples = 50000
	defaultLookback  = 7 * 24 * time.Hour
)

type route struct {
	Method      string
	Path        string
	Description string
}

type managementRequest struct {
	Method string
	Path   string
	Query  url.Values
	Body   []byte
}

type managementResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
}

func managementRoutes() any {
	return map[string][]route{"routes": {
		{Method: http.MethodGet, Path: samplesRoute, Description: "List quota samples since a unix ms timestamp."},
		{Method: http.MethodPost, Path: samplesRoute, Description: "Record quota samples."},
	}}
}

func (p *Plugin) handleManagement(raw []byte) ([]byte, bool) {
	var request managementRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		return failure("invalid_request", err.Error())
	}
	return success(p.serve(request))
}

func (p *Plugin) serve(request managementRequest) managementResponse {
	store := p.currentStore()
	switch {
	case request.Path != samplesPath:
		return errorResponse(http.StatusNotFound, "no such route")
	case store == nil:
		return errorResponse(http.StatusServiceUnavailable, "quota ledger is not configured")
	case request.Method == http.MethodGet:
		return p.listSamples(store, request.Query)
	case request.Method == http.MethodPost:
		return p.recordSamples(store, request.Body)
	default:
		return errorResponse(http.StatusNotFound, "no such route")
	}
}

func (p *Plugin) listSamples(store *ledger.Store, query url.Values) managementResponse {
	since := p.now().Add(-defaultLookback).UnixMilli()
	if raw := query.Get("since"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return errorResponse(http.StatusBadRequest, "since must be a unix ms timestamp")
		}
		since = parsed
	}
	return jsonResponse(http.StatusOK, map[string][]ledger.Sample{"samples": store.Since(since, maxServedSamples)})
}

func (p *Plugin) recordSamples(store *ledger.Store, body []byte) managementResponse {
	samples, err := p.parseSamples(body)
	if err != nil {
		return errorResponse(http.StatusBadRequest, err.Error())
	}
	accepted, skipped, err := store.Append(samples)
	if err != nil {
		return errorResponse(http.StatusInternalServerError, "store samples: "+err.Error())
	}
	return jsonResponse(http.StatusOK, map[string]int{"accepted": accepted, "skipped": skipped})
}

func (p *Plugin) parseSamples(body []byte) ([]ledger.Sample, error) {
	var payload struct {
		Samples *[]ledger.Sample `json:"samples"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	if payload.Samples == nil {
		return nil, fmt.Errorf("samples is required")
	}
	samples := *payload.Samples
	if len(samples) > maxPostedSamples {
		return nil, fmt.Errorf("send at most %d samples per request", maxPostedSamples)
	}
	now := p.now()
	for i, sample := range samples {
		if err := sample.Validate(now); err != nil {
			return nil, fmt.Errorf("samples[%d]: %w", i, err)
		}
	}
	return samples, nil
}

func errorResponse(status int, message string) managementResponse {
	return jsonResponse(status, map[string]string{"error": message})
}

func jsonResponse(status int, body any) managementResponse {
	raw, err := json.Marshal(body)
	if err != nil {
		status, raw = http.StatusInternalServerError, []byte(`{"error":"encode response"}`)
	}
	return managementResponse{
		StatusCode: status,
		Headers:    http.Header{"Content-Type": {"application/json"}},
		Body:       raw,
	}
}
