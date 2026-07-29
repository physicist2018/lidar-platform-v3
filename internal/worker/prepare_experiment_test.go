package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

func TestMedianFilter3(t *testing.T) {
	tests := []struct {
		name     string
		input    []float32
		expected []float32
	}{
		{"empty", nil, nil},
		{"single", []float32{42}, []float32{42}},
		{"two", []float32{10, 20}, []float32{15, 15}},
		{"three_asc", []float32{10, 20, 30}, []float32{15, 20, 25}},
		{"three_desc", []float32{30, 20, 10}, []float32{25, 20, 15}},
		{"spike_middle", []float32{10, 100, 20}, []float32{55, 20, 60}},
		{"spike_start", []float32{100, 10, 20}, []float32{55, 20, 15}},
		{"spike_end", []float32{10, 20, 100}, []float32{15, 20, 60}},
		{"five", []float32{100, 200, 300, 400, 500}, []float32{150, 200, 300, 400, 450}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := medianFilter3(tt.input)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestProcessProfile_BackgroundFile(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	signal := []float64{100, 200, 300, 400, 500}
	bg := []float64{10, 20, 30, 40, 50}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      signal,
			BinWidth:  30,
		},
		Background: &domain.ProfileData{
			Data: bg,
		},
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{
		BackgroundType: "file",
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Equal(t, meta.ID, result.PreparedMetaID)
	assert.Equal(t, pp.Signal.ProfileID, result.LicelProfileID)

	expected := []float32{90, 180, 270, 360, 450}
	assert.Equal(t, expected, result.Data)
}

func TestProcessProfile_BackgroundFile_ShorterBackground(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	signal := []float64{100, 200, 300, 400, 500}
	bg := []float64{10, 20, 30}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      signal,
			BinWidth:  30,
		},
		Background: &domain.ProfileData{
			Data: bg,
		},
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{
		BackgroundType: "file",
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Len(t, result.Data, 5)
	assert.Equal(t, float32(90), result.Data[0])
	assert.Equal(t, float32(400), result.Data[3])

	payload.TrimFrom = 60
	result, err = handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Len(t, result.Data, 2)
}

func TestProcessProfile_BackgroundMean(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	// signal: [100, 200, 300, 400, 500]
	// median filtered: [150, 200, 300, 400, 450]
	// tail from index 3: [400, 450], mean = 425
	signal := []float64{100, 200, 300, 400, 500}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      signal,
			BinWidth:  30,
		},
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{
		BackgroundType: "mean",
		BackgroundFrom: 90,
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundMean, payload)
	assert.NoError(t, err)
	// [100-425, 200-425, 300-425, 400-425, 500-425] = [-325, -225, -125, -25, 75]
	assert.Equal(t, []float32{-325, -225, -125, -25, 75}, result.Data)
}

func TestProcessProfile_BackgroundMean_TailStartOutOfRange(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	signal := []float64{100, 200}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      signal,
			BinWidth:  30,
		},
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{
		BackgroundType: "mean",
		BackgroundFrom: 300,
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundMean, payload)
	assert.NoError(t, err)
	// tailStart >= len(result) -> len/2 = 1
	// median filtered: [150, 150], tail from 1: [150], mean = 150
	// [100-150, 200-150] = [-50, 50]
	assert.Equal(t, []float32{-50, 50}, result.Data)
}

func TestProcessProfile_NoBackground(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	signal := []float64{100, 200, 300}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      signal,
			BinWidth:  30,
		},
		Background: nil,
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{
		BackgroundType: "file",
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Equal(t, []float32{100, 200, 300}, result.Data)
}

func TestProcessProfile_InvalidBinWidth(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	pp := domain.PairedProfile{
		Signal: domain.ProfileData{
			ProfileID: uuid.New(),
			Data:      []float64{100},
			BinWidth:  0,
		},
	}

	meta := domain.PreparedMeta{ID: uuid.New()}
	payload := PrepareExperimentPayload{BackgroundType: "file", TrimFrom: 100}

	_, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bin_width")
}
