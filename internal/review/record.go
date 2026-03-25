package review

import (
	"encoding/hex"
	"image/color"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
)

func NewRecord(id string, createdAt time.Time, cfg core.Config, result *core.Result, fingerprints Fingerprints, artifacts ArtifactManifest) (ReviewRecord, error) {
	normalized, err := core.NormalizeConfig(cfg)
	if err != nil {
		return ReviewRecord{}, err
	}

	record := ReviewRecord{
		ID:           id,
		CreatedAt:    createdAt.UTC(),
		Mode:         string(normalized.Mode),
		Config:       normalized,
		Source:       result.SourceMeta,
		OutputWidth:  result.FinalImage.Bounds().Dx(),
		OutputHeight: result.FinalImage.Bounds().Dy(),
		Artifacts:    fillArtifactDefaults(artifacts),
		Fingerprints: fingerprints,
		Metadata:     cloneMetadata(result.Metadata),
	}

	if len(result.GlobalPalette) > 0 {
		record.GlobalPalette = colorsToHex(result.GlobalPalette)
	}
	if len(result.PaletteBanks) > 0 {
		record.PaletteBanks = make([][]string, 0, len(result.PaletteBanks))
		for _, bank := range result.PaletteBanks {
			record.PaletteBanks = append(record.PaletteBanks, colorsToHex(bank.Colors))
		}
	}
	if len(result.TileAssignments) > 0 {
		record.TileAssignments = append([]core.TileAssignment(nil), result.TileAssignments...)
	}

	return record, nil
}

func fillArtifactDefaults(artifacts ArtifactManifest) ArtifactManifest {
	if artifacts.FinalPNG == "" {
		artifacts.FinalPNG = DefaultFinalPNGName
	}
	if artifacts.PreviewPNG == "" {
		artifacts.PreviewPNG = DefaultPreviewPNGName
	}
	if artifacts.MetaJSON == "" {
		artifacts.MetaJSON = DefaultMetaJSONName
	}
	return artifacts
}

func colorsToHex(colors []color.NRGBA) []string {
	out := make([]string, 0, len(colors))
	for _, c := range colors {
		out = append(out, colorHex(c))
	}
	return out
}

func colorHex(c color.NRGBA) string {
	buf := []byte{c.R, c.G, c.B}
	return "#" + hex.EncodeToString(buf)
}

func cloneMetadata(metadata map[string]any) map[string]any {
	if len(metadata) == 0 {
		return nil
	}

	out := make(map[string]any, len(metadata))
	for key, value := range metadata {
		out[key] = value
	}
	return out
}
