package repository

import (
	"github.com/physcist2018/lidar-platform-v3/internal/lidar/domain"
	db "github.com/physcist2018/lidar-platform-v3/pkg/db/lidar"
)

// MapPairedProfile converts a sqlc query row to a domain PairedProfile.
func MapPairedProfile(row db.FindProfilesWithBackgroundByExperimentRow) domain.PairedProfile {
	var bg *domain.ProfileData
	if row.BgProfileID.Valid {
		bg = &domain.ProfileData{
			ProfileID:   row.BgProfileID.UUID,
			LicelFileID: row.BgLicelfileID.UUID,
			Data:        row.BgData,
			NumPoints:   row.BgPoints.Int32,
			BinWidth:    float32(row.BgBinWidth.Float64),
		}
	}

	return domain.PairedProfile{
		ExperimentID: row.ExperimentID,
		DeviceID:     row.DeviceID,
		Wavelength:   row.Wavelength,
		Polarization: row.Polarization,
		Signal: domain.ProfileData{
			ProfileID:   row.SignalProfileID,
			LicelFileID: row.SignalLicelfileID,
			Data:        row.SignalData,
			NumPoints:   row.SignalPoints,
			BinWidth:    row.SignalBinWidth,
		},
		Background:  bg,
		MatchStatus: domain.MatchStatus(row.DataLengthMatch),
	}
}
