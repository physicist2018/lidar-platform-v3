package ycf

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/pkg/lidar/molecular"
)

func validBody(t *testing.T) []byte {
	in := molecular.Input{
		Range:       []float64{0, 100, 200},
		ZenithAngle: 0,
		Wavelength:  532,
		Atmosphere: molecular.AtmosphereModel{
			Altitude:    []float64{0, 5},
			Temperature: []float64{15, -20},
			Pressure:    []float64{1013, 540},
		},
	}
	b, err := json.Marshal(in)
	require.NoError(t, err)
	return b
}

func doRequest(t *testing.T, method string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, "/", bytes.NewReader(body))
	w := httptest.NewRecorder()
	Handler(w, req)
	return w
}

func TestHandler_Success(t *testing.T) {
	w := doRequest(t, http.MethodPost, validBody(t))
	require.Equal(t, http.StatusOK, w.Code)

	var res molecular.Result
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))

	require.Len(t, res.Backscatter, 3)
	require.Len(t, res.Extinction, 3)
	require.Len(t, res.Transmission, 3)
	require.Len(t, res.Signal, 3)

	// Signal = backscatter · transmission (range-corrected).
	for i := range res.Signal {
		assert.InDelta(t, res.Backscatter[i]*res.Transmission[i], res.Signal[i], 1e-15, "point %d", i)
	}
	// At r = 0 the transmission is 1.
	assert.InDelta(t, 1, res.Transmission[0], 1e-15)
}

func TestHandler_SuccessWithSnakeCaseJSON(t *testing.T) {
	// The request uses the snake_case json field names.
	body := `{
		"range": [0, 500],
		"zenith_angle": 0,
		"wavelength": 532,
		"atmosphere": {
			"altitude": [0, 10],
			"temperature": [15, -50],
			"pressure": [1013, 265]
		}
	}`
	w := doRequest(t, http.MethodPost, []byte(body))
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_InvalidJSON(t *testing.T) {
	w := doRequest(t, http.MethodPost, []byte(`{not-json`))
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestHandler_ValidationError(t *testing.T) {
	// Wavelength below the Edlén validity range → validation error (400).
	body := `{
		"range": [0, 100],
		"zenith_angle": 0,
		"wavelength": 100,
		"atmosphere": {"altitude": [0, 5], "temperature": [15, -20], "pressure": [1013, 540]}
	}`
	w := doRequest(t, http.MethodPost, []byte(body))
	require.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "wavelength")
}

func TestHandler_MethodNotAllowed(t *testing.T) {
	w := doRequest(t, http.MethodGet, nil)
	require.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestHandler_CORS(t *testing.T) {
	w := doRequest(t, http.MethodPost, validBody(t))
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	assert.Contains(t, w.Header().Get("Access-Control-Allow-Methods"), "POST")

	// OPTIONS preflight → 200 without processing.
	pre := doRequest(t, http.MethodOptions, nil)
	require.Equal(t, http.StatusOK, pre.Code)
	assert.Equal(t, "*", pre.Header().Get("Access-Control-Allow-Origin"))
}
