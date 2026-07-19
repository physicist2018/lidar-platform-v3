package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/application"
)

// ---------------------------------------------------------------------------
// Mock
// ---------------------------------------------------------------------------

type mockCreateExperimentUC struct {
	executeFunc func(ctx context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error)
}

func (m *mockCreateExperimentUC) Execute(ctx context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error) {
	if m.executeFunc == nil {
		panic("mockCreateExperimentUC.Execute called unexpectedly")
	}
	return m.executeFunc(ctx, req)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newMockHandler(fn func(ctx context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error)) *ExperimentHandler {
	return NewExperimentHandler(&mockCreateExperimentUC{executeFunc: fn})
}

func buildMultipart(t *testing.T, fields map[string]string, files map[string]string) (body bytes.Buffer, contentType string) {
	t.Helper()
	w := multipart.NewWriter(&body)
	defer w.Close()

	for key, val := range fields {
		require.NoError(t, w.WriteField(key, val))
	}

	for field, content := range files {
		if content == "" {
			continue
		}
		fw, err := w.CreateFormFile(field, field+".zip")
		require.NoError(t, err)
		_, err = io.WriteString(fw, content)
		require.NoError(t, err)
	}

	return body, w.FormDataContentType()
}

type testCase struct {
	name       string
	fields     map[string]string
	files      map[string]string
	mockFn     func(ctx context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error)
	wantStatus int
	wantBody   map[string]any
}

func runTestCase(t *testing.T, tc testCase) {
	t.Helper()

	var handler *ExperimentHandler
	if tc.mockFn != nil {
		handler = newMockHandler(tc.mockFn)
	} else {
		handler = newMockHandler(nil)
	}

	body, contentType := buildMultipart(t, tc.fields, tc.files)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/experiments/create", &body)
	req.Header.Set("Content-Type", contentType)

	w := httptest.NewRecorder()
	handler.HandleCreateExperiment(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	assert.Equal(t, tc.wantStatus, resp.StatusCode)

	var respBody map[string]any
	err := json.NewDecoder(resp.Body).Decode(&respBody)
	require.NoError(t, err)

	for k, v := range tc.wantBody {
		assert.Equal(t, v, respBody[k], "field %q", k)
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestHandleCreateExperiment_Success(t *testing.T) {
	fields := map[string]string{
		"title":        "Test Experiment",
		"zenith_angle": "45.5",
		"latitude":     "43.1",
		"longitude":    "131.9",
		"comments":     "test comments",
	}
	files := map[string]string{
		"experiment_files": "zip-content",
		"background":       "bg-data",
		"meteo":            "weather-data",
	}
	mockFn := func(_ context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error) {
		assert.Equal(t, "Test Experiment", req.Title)
		assert.InDelta(t, 45.5, req.ZenithAngle, 0.01)
		assert.InDelta(t, 43.1, req.Latitude, 0.01)
		assert.InDelta(t, 131.9, req.Longitude, 0.01)
		assert.Equal(t, "test comments", req.Comments)
		assert.Equal(t, "experiment_files.zip", req.ExperimentFiles.Filename)
		assert.NotNil(t, req.Background)
		assert.Equal(t, "background.zip", req.Background.Filename)
		assert.NotNil(t, req.Meteo)
		assert.Equal(t, "meteo.zip", req.Meteo.Filename)
		return &application.CreateExperimentResponse{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Title:       "Test Experiment",
			ZenithAngle: 45.5,
			Latitude:    43.1,
			Longitude:   131.9,
			Comments:    "test comments",
		}, nil
	}

	runTestCase(t, testCase{
		name:       "all fields and files",
		fields:     fields,
		files:      files,
		mockFn:     mockFn,
		wantStatus: http.StatusCreated,
		wantBody: map[string]any{
			"id":           "00000000-0000-0000-0000-000000000001",
			"title":        "Test Experiment",
			"zenith_angle": float64(45.5),
			"latitude":     float64(43.1),
			"longitude":    float64(131.9),
			"comments":     "test comments",
		},
	})
}

func TestHandleCreateExperiment_Success_NoOptionalFiles(t *testing.T) {
	fields := map[string]string{
		"title":        "Minimal",
		"zenith_angle": "30",
		"latitude":     "0",
		"longitude":    "0",
	}
	files := map[string]string{
		"experiment_files": "data",
	}
	mockFn := func(_ context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error) {
		assert.Nil(t, req.Background)
		assert.Nil(t, req.Meteo)
		return &application.CreateExperimentResponse{
			ID:          uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Title:       "Minimal",
			ZenithAngle: 30,
			Latitude:    0,
			Longitude:   0,
		}, nil
	}

	runTestCase(t, testCase{
		name:       "only required fields and files",
		fields:     fields,
		files:      files,
		mockFn:     mockFn,
		wantStatus: http.StatusCreated,
		wantBody: map[string]any{
			"id":           "00000000-0000-0000-0000-000000000002",
			"title":        "Minimal",
			"zenith_angle": float64(30),
			"latitude":     float64(0),
			"longitude":    float64(0),
		},
	})
}

func TestHandleCreateExperiment_MissingTitle(t *testing.T) {
	runTestCase(t, testCase{
		name: "missing title",
		fields: map[string]string{
			"zenith_angle": "30",
			"latitude":     "0",
			"longitude":    "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "title is required"},
	})
}

func TestHandleCreateExperiment_MissingZenithAngle(t *testing.T) {
	runTestCase(t, testCase{
		name: "missing zenith_angle",
		fields: map[string]string{
			"title":     "Test",
			"latitude":  "0",
			"longitude": "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "zenith_angle is required and must be a number"},
	})
}

func TestHandleCreateExperiment_InvalidZenithAngle(t *testing.T) {
	runTestCase(t, testCase{
		name: "invalid zenith_angle",
		fields: map[string]string{
			"title":        "Test",
			"zenith_angle": "not-a-number",
			"latitude":     "0",
			"longitude":    "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "zenith_angle is required and must be a number"},
	})
}

func TestHandleCreateExperiment_MissingLatitude(t *testing.T) {
	runTestCase(t, testCase{
		name: "missing latitude",
		fields: map[string]string{
			"title":        "Test",
			"zenith_angle": "30",
			"longitude":    "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "latitude is required and must be a number"},
	})
}

func TestHandleCreateExperiment_MissingLongitude(t *testing.T) {
	runTestCase(t, testCase{
		name: "missing longitude",
		fields: map[string]string{
			"title":        "Test",
			"zenith_angle": "30",
			"latitude":     "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "longitude is required and must be a number"},
	})
}

func TestHandleCreateExperiment_MissingExperimentFiles(t *testing.T) {
	runTestCase(t, testCase{
		name: "missing experiment_files",
		fields: map[string]string{
			"title":        "Test",
			"zenith_angle": "30",
			"latitude":     "0",
			"longitude":    "0",
		},
		files:      nil,
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "experiment_files is required"},
	})
}

func TestHandleCreateExperiment_UseCaseError(t *testing.T) {
	fields := map[string]string{
		"title":        "Test",
		"zenith_angle": "30",
		"latitude":     "0",
		"longitude":    "0",
	}
	files := map[string]string{
		"experiment_files": "data",
	}
	mockFn := func(_ context.Context, req *application.CreateExperimentRequest) (*application.CreateExperimentResponse, error) {
		return nil, errors.New("something went wrong")
	}

	runTestCase(t, testCase{
		name:       "use case returns error",
		fields:     fields,
		files:      files,
		mockFn:     mockFn,
		wantStatus: http.StatusInternalServerError,
		wantBody:   map[string]any{"error": "internal server error"},
	})
}

func TestHandleCreateExperiment_EmptyTitle(t *testing.T) {
	runTestCase(t, testCase{
		name: "empty title",
		fields: map[string]string{
			"title":        "",
			"zenith_angle": "30",
			"latitude":     "0",
			"longitude":    "0",
		},
		files: map[string]string{
			"experiment_files": "data",
		},
		mockFn:     nil,
		wantStatus: http.StatusBadRequest,
		wantBody:   map[string]any{"error": "title is required"},
	})
}
