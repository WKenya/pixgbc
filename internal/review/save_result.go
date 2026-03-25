package review

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/WKenya/pixgbc/internal/core"
	"github.com/WKenya/pixgbc/internal/export"
	"github.com/WKenya/pixgbc/internal/ioimg"
)

func SaveResult(ctx context.Context, store Store, inputBytes []byte, cfg core.Config, result *core.Result) (ReviewRecord, error) {
	finalPNG, err := export.PNGBytes(result.FinalImage)
	if err != nil {
		return ReviewRecord{}, err
	}
	previewPNG, err := export.PNGBytes(result.PreviewImage)
	if err != nil {
		return ReviewRecord{}, err
	}

	inputHash := HashBytes(inputBytes)
	configHash, err := HashConfig(cfg)
	if err != nil {
		return ReviewRecord{}, err
	}

	now := time.Now().UTC()
	record, err := NewRecord(makeReviewID(now, inputHash), now, cfg, result, Fingerprints{
		InputSHA256:  inputHash,
		ConfigSHA256: configHash,
		OutputSHA256: HashBytes(finalPNG),
	}, ArtifactManifest{})
	if err != nil {
		return ReviewRecord{}, err
	}

	files := map[string][]byte{
		record.Artifacts.FinalPNG:   finalPNG,
		record.Artifacts.PreviewPNG: previewPNG,
	}
	if cfg.EmitDebug {
		decoded, err := ioimg.DecodeImage(bytes.NewReader(inputBytes), ioimg.DefaultLimits())
		if err != nil {
			return ReviewRecord{}, err
		}
		debugPNG, err := export.DebugSheetPNG(decoded.Image, result)
		if err != nil {
			return ReviewRecord{}, err
		}
		record.Artifacts.DebugPNG = DefaultDebugPNGName
		files[record.Artifacts.DebugPNG] = debugPNG
	}

	if err := store.Save(ctx, record, files); err != nil {
		return ReviewRecord{}, err
	}

	return record, nil
}

func makeReviewID(now time.Time, inputHash string) string {
	prefix := inputHash
	if len(prefix) > 12 {
		prefix = prefix[:12]
	}
	return fmt.Sprintf("%d-%s", now.UnixNano(), prefix)
}
