package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTaskRecord_Pending(t *testing.T) {
	id := uuid.New()
	subject := "lidar.task.test"
	expID := uuid.New()

	rec := NewTaskRecord(id, subject, &expID, nil)

	assert.Equal(t, id, rec.ID)
	assert.Equal(t, subject, rec.Subject)
	assert.Equal(t, TaskPending, rec.Status)
	assert.Equal(t, &expID, rec.ExperimentID)
	assert.NotZero(t, rec.CreatedAt)
	assert.NotZero(t, rec.UpdatedAt)
	assert.Nil(t, rec.StartedAt)
	assert.Nil(t, rec.FinishedAt)
	assert.Empty(t, rec.ErrorMessage)
}

func TestNewTaskRecord_NilExperimentID(t *testing.T) {
	id := uuid.New()
	rec := NewTaskRecord(id, "lidar.task.profile", nil, nil)
	assert.Equal(t, id, rec.ID)
	assert.Nil(t, rec.ExperimentID)
}

func TestNewTaskRecord_NilParams(t *testing.T) {
	rec := NewTaskRecord(uuid.New(), "lidar.task.test", nil, nil)
	assert.Nil(t, rec.TaskParams)
}

func TestNewExperiment_Defaults(t *testing.T) {
	loc, err := NewGeoLocation(43.1, 131.9)
	require.NoError(t, err)
	tr, err := NewTimeRange(
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC),
	)
	require.NoError(t, err)

	exp := NewExperiment("Test Experiment", 45.5, tr, loc)
	assert.Equal(t, "Test Experiment", exp.Title)
	assert.Equal(t, float32(45.5), exp.ZenithAngle)
	assert.NotZero(t, exp.ID)
}

func TestNewExperiment_WithOptions(t *testing.T) {
	loc, _ := NewGeoLocation(0, 0)
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	refs := ExperimentStorageRefs{ExperimentDataID: uuidPtr()}

	exp := NewExperiment("Test", 30, tr, loc, WithComments("notes"), WithStorageRefs(refs))
	assert.Equal(t, "notes", exp.Comments)
	assert.Equal(t, refs, exp.StorageRefs)
}

func TestNewTimeRange_Valid(t *testing.T) {
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)

	tr, err := NewTimeRange(start, end)
	require.NoError(t, err)
	assert.Equal(t, time.Hour, tr.Duration())
}

func TestNewTimeRange_Invalid(t *testing.T) {
	start := time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC)
	end := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	_, err := NewTimeRange(start, end)
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestNewTimeRange_EqualTimes(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err := NewTimeRange(now, now)
	assert.ErrorIs(t, err, ErrInvalidTimeRange)
}

func TestNewGeoLocation_Valid(t *testing.T) {
	loc, err := NewGeoLocation(43.1, 131.9)
	require.NoError(t, err)
	assert.InDelta(t, float32(43.1), loc.Latitude, 0.01)
	assert.InDelta(t, float32(131.9), loc.Longitude, 0.01)
}

func TestNewGeoLocation_InvalidLatitude(t *testing.T) {
	_, err := NewGeoLocation(100, 0)
	assert.ErrorIs(t, err, ErrInvalidGeoLocation)
	_, err = NewGeoLocation(-100, 0)
	assert.ErrorIs(t, err, ErrInvalidGeoLocation)
}

func TestNewGeoLocation_InvalidLongitude(t *testing.T) {
	_, err := NewGeoLocation(0, 200)
	assert.ErrorIs(t, err, ErrInvalidGeoLocation)
	_, err = NewGeoLocation(0, -200)
	assert.ErrorIs(t, err, ErrInvalidGeoLocation)
}

func TestExperiment_SoftDeleteAndRestore(t *testing.T) {
	loc, _ := NewGeoLocation(0, 0)
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	exp := NewExperiment("Test", 30, tr, loc)

	exp.SoftDelete()
	assert.NotNil(t, exp.DeletedAt)
	assert.True(t, exp.IsDeleted())

	exp.Restore()
	assert.Nil(t, exp.DeletedAt)
	assert.False(t, exp.IsDeleted())
}

func TestNewLicelFile_Basic(t *testing.T) {
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	lf := NewLicelFile(uuid.New(), tr, 5, 100, false, uuid.New())

	assert.Equal(t, int32(5), lf.NDatasets)
	assert.Equal(t, int32(100), lf.LaserFreq)
	assert.False(t, lf.IsBackground)
	assert.NotZero(t, lf.ID)
}

func TestNewLicelFile_WithFilename(t *testing.T) {
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	lf := NewLicelFile(uuid.New(), tr, 1, 10, true, uuid.New(), WithFilename("test.licel"))
	assert.True(t, lf.IsBackground)
	assert.Equal(t, "test.licel", lf.Filename)
}

func TestLicelFile_SoftDeleteAndRestore(t *testing.T) {
	tr, _ := NewTimeRange(time.Now(), time.Now().Add(time.Hour))
	lf := NewLicelFile(uuid.New(), tr, 1, 10, false, uuid.New())

	lf.SoftDelete()
	assert.NotNil(t, lf.DeletedAt)
	assert.True(t, lf.IsDeleted())

	lf.Restore()
	assert.Nil(t, lf.DeletedAt)
	assert.False(t, lf.IsDeleted())
}

func TestNewLicelProfile_Valid(t *testing.T) {
	data := []float64{1.0, 2.0, 3.0}
	p, err := NewLicelProfile(uuid.New(), 3, 500.0, 0.1, 532.0, "P", "dev1", 100, 0.05, data)
	require.NoError(t, err)
	assert.Equal(t, data, p.Data)
}

func TestNewLicelProfile_DataMismatch(t *testing.T) {
	_, err := NewLicelProfile(uuid.New(), 5, 500, 0.1, 532, "P", "dev1", 100, 0.05, []float64{1.0, 2.0})
	assert.ErrorIs(t, err, ErrProfileDataMismatch)
}

func TestLicelProfile_PointAt(t *testing.T) {
	p, _ := NewLicelProfile(uuid.New(), 3, 500, 0.1, 532, "P", "dev1", 100, 0.05, []float64{10, 20, 30})
	v, err := p.PointAt(0)
	assert.NoError(t, err)
	assert.Equal(t, float64(10), v)
	_, err = p.PointAt(5)
	assert.Error(t, err)
}

func TestNewAtmosphereProfile_Valid(t *testing.T) {
	p, err := NewAtmosphereProfile(uuid.New(), []float64{0.5, 1.0}, []float64{300, 295}, []float64{100000, 90000})
	require.NoError(t, err)
	assert.Equal(t, 2, p.NumPoints())
}

func TestNewAtmosphereProfile_LengthMismatch(t *testing.T) {
	_, err := NewAtmosphereProfile(uuid.New(), []float64{1, 2}, []float64{1}, []float64{1})
	assert.ErrorIs(t, err, ErrProfileDataMismatch)
}

func TestObjectPath_Key(t *testing.T) {
	p := ObjectPath{Bucket: "mybucket", Path: "path/to/file.txt"}
	assert.Equal(t, "mybucket/path/to/file.txt", p.Key())
	assert.Equal(t, p.Key(), p.String())
}

func TestNewObjectPath_Validation(t *testing.T) {
	_, err := NewObjectPath("", "path")
	assert.ErrorIs(t, err, ErrInvalidPath)
	_, err = NewObjectPath("bucket", "")
	assert.ErrorIs(t, err, ErrInvalidPath)
}

func TestNewStorageObject(t *testing.T) {
	p, _ := NewObjectPath("bucket", "path")
	obj := NewStorageObject(p, WithSize(100), WithETag("abc"), WithContentType("text/plain"))
	assert.Equal(t, int64(100), obj.Size)
	assert.Equal(t, "abc", obj.ETag)
	assert.Equal(t, "text/plain", obj.ContentType)
}

func uuidPtr() *uuid.UUID {
	id := uuid.New()
	return &id
}
