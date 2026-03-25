package palette

import (
	"fmt"
	"image/color"
	"slices"
	"strings"
)

type bankCluster struct {
	counts map[uint16]int
	colors []color.NRGBA
}

func ClusterTilePalettes(tilePalettes [][]color.NRGBA, maxBanks int, colorsPerTile int) [][]color.NRGBA {
	if len(tilePalettes) == 0 || maxBanks <= 0 || colorsPerTile <= 0 {
		return nil
	}

	unique := make(map[string]*bankCluster)
	for _, palette := range tilePalettes {
		normalized := normalizePaletteColors(palette, colorsPerTile)
		key := paletteKey(normalized)
		cluster, ok := unique[key]
		if !ok {
			cluster = &bankCluster{
				counts: paletteCounts(normalized),
				colors: normalized,
			}
			unique[key] = cluster
			continue
		}
		for bucket, count := range paletteCounts(normalized) {
			cluster.counts[bucket] += count
		}
		cluster.colors = representativeColors(cluster.counts, colorsPerTile)
	}

	banks := make([]*bankCluster, 0, len(unique))
	for _, cluster := range unique {
		banks = append(banks, cluster)
	}
	sortClusters(banks)

	for len(banks) > maxBanks {
		left, right := nearestClusterPair(banks)
		merged := mergeClusters(banks[left], banks[right], colorsPerTile)
		next := make([]*bankCluster, 0, len(banks)-1)
		for i, cluster := range banks {
			if i == left {
				next = append(next, merged)
				continue
			}
			if i == right {
				continue
			}
			next = append(next, cluster)
		}
		banks = next
		sortClusters(banks)
	}

	out := make([][]color.NRGBA, 0, len(banks))
	for _, bank := range banks {
		out = append(out, append([]color.NRGBA(nil), bank.colors...))
	}

	return out
}

func AssignTilePalettesToBanks(tilePalettes [][]color.NRGBA, banks [][]color.NRGBA) []int {
	assignments := make([]int, 0, len(tilePalettes))
	for _, tilePalette := range tilePalettes {
		assignment := 0
		bestDistance := paletteDistance(normalizePaletteColors(tilePalette, len(banks[0])), banks[0])
		for i := 1; i < len(banks); i++ {
			distance := paletteDistance(tilePalette, banks[i])
			if distance < bestDistance {
				bestDistance = distance
				assignment = i
			}
		}
		assignments = append(assignments, assignment)
	}
	return assignments
}

func paletteDistance(a, b []color.NRGBA) int {
	total := 0
	for _, colorA := range a {
		total += nearestColorDistance(colorA, b)
	}
	for _, colorB := range b {
		total += nearestColorDistance(colorB, a)
	}
	return total
}

func nearestColorDistance(target color.NRGBA, palette []color.NRGBA) int {
	best := SquaredDistance(target, palette[0])
	for i := 1; i < len(palette); i++ {
		distance := SquaredDistance(target, palette[i])
		if distance < best {
			best = distance
		}
	}
	return best
}

func mergeClusters(a, b *bankCluster, colorsPerTile int) *bankCluster {
	counts := make(map[uint16]int, len(a.counts)+len(b.counts))
	for bucket, count := range a.counts {
		counts[bucket] += count
	}
	for bucket, count := range b.counts {
		counts[bucket] += count
	}
	return &bankCluster{
		counts: counts,
		colors: representativeColors(counts, colorsPerTile),
	}
}

func representativeColors(counts map[uint16]int, colorsPerTile int) []color.NRGBA {
	type entry struct {
		bucket uint16
		count  int
	}

	entries := make([]entry, 0, len(counts))
	for bucket, count := range counts {
		entries = append(entries, entry{bucket: bucket, count: count})
	}

	slices.SortFunc(entries, func(a, b entry) int {
		if a.count != b.count {
			return b.count - a.count
		}
		if a.bucket < b.bucket {
			return -1
		}
		if a.bucket > b.bucket {
			return 1
		}
		return 0
	})

	colors := make([]color.NRGBA, 0, colorsPerTile)
	for _, entry := range entries {
		if len(colors) >= colorsPerTile {
			break
		}
		colors = append(colors, ExpandRGB555(entry.bucket))
	}
	for len(colors) < colorsPerTile && len(colors) > 0 {
		colors = append(colors, colors[len(colors)-1])
	}
	slices.SortFunc(colors, compareByLuma)
	return colors
}

func paletteCounts(colors []color.NRGBA) map[uint16]int {
	counts := make(map[uint16]int, len(colors))
	for _, c := range colors {
		counts[ReduceRGB555(c)]++
	}
	return counts
}

func normalizePaletteColors(colors []color.NRGBA, colorsPerTile int) []color.NRGBA {
	normalized := append([]color.NRGBA(nil), colors...)
	slices.SortFunc(normalized, compareByLuma)
	for len(normalized) < colorsPerTile && len(normalized) > 0 {
		normalized = append(normalized, normalized[len(normalized)-1])
	}
	if len(normalized) > colorsPerTile {
		normalized = normalized[:colorsPerTile]
	}
	return normalized
}

func paletteKey(colors []color.NRGBA) string {
	parts := make([]string, 0, len(colors))
	for _, c := range colors {
		parts = append(parts, strings.ToLower(colorKey(c)))
	}
	return strings.Join(parts, ",")
}

func colorKey(c color.NRGBA) string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

func sortClusters(clusters []*bankCluster) {
	slices.SortFunc(clusters, func(a, b *bankCluster) int {
		keyA := paletteKey(a.colors)
		keyB := paletteKey(b.colors)
		if keyA < keyB {
			return -1
		}
		if keyA > keyB {
			return 1
		}
		return 0
	})
}

func nearestClusterPair(clusters []*bankCluster) (int, int) {
	bestLeft, bestRight := 0, 1
	bestDistance := paletteDistance(clusters[0].colors, clusters[1].colors)
	bestKey := paletteKey(clusters[0].colors) + "|" + paletteKey(clusters[1].colors)

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			distance := paletteDistance(clusters[i].colors, clusters[j].colors)
			key := paletteKey(clusters[i].colors) + "|" + paletteKey(clusters[j].colors)
			if distance < bestDistance || (distance == bestDistance && key < bestKey) {
				bestDistance = distance
				bestKey = key
				bestLeft = i
				bestRight = j
			}
		}
	}

	return bestLeft, bestRight
}
