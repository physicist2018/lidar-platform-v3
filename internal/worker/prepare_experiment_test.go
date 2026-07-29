package worker

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
)

func TestProcessProfile_BackgroundFile(t *testing.T) {
	handler := &PrepareExperimentHandler{} // only uses processProfile — no dependencies needed

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
		TrimFrom:       150, // 150m / 30m per bin = 5 bins
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Equal(t, meta.ID, result.PreparedMetaID)
	assert.Equal(t, pp.Signal.ProfileID, result.LicelProfileID)

	// Expected: signal - background = [90, 180, 270, 360, 450]
	expected := []float32{90, 180, 270, 360, 450}
	assert.Equal(t, expected, result.Data)
}

func TestProcessProfile_BackgroundFile_ShorterBackground(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	signal := []float64{100, 200, 300, 400, 500}
	bg := []float64{10, 20, 30} // shorter than signal

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
	// Only first 3 bins subtracted, rest unchanged before trim
	// Expected: [90, 180, 270, 400, 500] trimmed to 5 bins
	assert.Len(t, result.Data, 5)
	assert.Equal(t, float32(90), result.Data[0])
	assert.Equal(t, float32(400), result.Data[3])

	// Verify trim
	payload.TrimFrom = 60 // 60/30 = 2 bins
	result, err = handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	assert.Len(t, result.Data, 2)
}

func TestProcessProfile_BackgroundMean(t *testing.T) {
	handler := &PrepareExperimentHandler{}

	// signal: tail from index 3 onward = [400, 500], mean = 450
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
		BackgroundFrom: 90,  // 90/30 = bin index 3
		TrimFrom:       150, // 150/30 = 5 bins
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundMean, payload)
	assert.NoError(t, err)
	// Expected: [100-450, 200-450, 300-450, 400-450, 500-450] = [-350, -250, -150, -50, 50]
	assert.Equal(t, []float32{-350, -250, -150, -50, 50}, result.Data)
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
		BackgroundFrom: 300, // 300/30 = 10 bins, larger than signal length
		TrimFrom:       150,
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundMean, payload)
	assert.NoError(t, err)
	// tailStart >= len(result) → default to len/2 = 1
	// mean of [200] = 200, so [100-200, 200-200] = [-100, 0]
	assert.Equal(t, []float32{-100, 0}, result.Data)
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
		TrimFrom:       150, // 5 bins, but signal has 3
	}

	result, err := handler.processProfile(pp, meta, domain.BackgroundFromFile, payload)
	assert.NoError(t, err)
	// No background file — signal unchanged, trim doesn't cut
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
