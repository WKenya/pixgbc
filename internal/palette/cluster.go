package palette

import (
	"image/color"
	"slices"
	"strings"
)

type bankCluster struct {
	counts map[uint16]int
	colors []color.NRGBA
	key    string
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
				key:    key,
			}
			unique[key] = cluster
			continue
		}
		for bucket, count := range paletteCounts(normalized) {
			cluster.counts[bucket] += count
		}
		cluster.colors = representativeColors(cluster.counts, colorsPerTile)
		cluster.key = paletteKey(cluster.colors)
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
	if len(tilePalettes) == 0 || len(banks) == 0 {
		return nil
	}

	assignments := make([]int, 0, len(tilePalettes))
	for _, tilePalette := range tilePalettes {
		assignment := 0
		bestDistance := PaletteDistance(normalizePaletteColors(tilePalette, len(banks[0])), banks[0])
		for i := 1; i < len(banks); i++ {
			distance := PaletteDistance(tilePalette, banks[i])
			if distance < bestDistance {
				bestDistance = distance
				assignment = i
			}
		}
		assignments = append(assignments, assignment)
	}
	return assignments
}

func PaletteDistance(a, b []color.NRGBA) int {
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
	colors := representativeColors(counts, colorsPerTile)
	return &bankCluster{
		counts: counts,
		colors: colors,
		key:    paletteKey(colors),
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
		parts = append(parts, colorKey(c))
	}
	return strings.Join(parts, ",")
}

func colorKey(c color.NRGBA) string {
	const hex = "0123456789abcdef"
	buf := [6]byte{
		hex[c.R>>4], hex[c.R&0x0F],
		hex[c.G>>4], hex[c.G&0x0F],
		hex[c.B>>4], hex[c.B&0x0F],
	}
	return string(buf[:])
}

func sortClusters(clusters []*bankCluster) {
	slices.SortFunc(clusters, func(a, b *bankCluster) int {
		if a.key < b.key {
			return -1
		}
		if a.key > b.key {
			return 1
		}
		return 0
	})
}

func nearestClusterPair(clusters []*bankCluster) (int, int) {
	bestLeft, bestRight := 0, 1
	bestDistance := PaletteDistance(clusters[0].colors, clusters[1].colors)

	for i := 0; i < len(clusters); i++ {
		for j := i + 1; j < len(clusters); j++ {
			distance := PaletteDistance(clusters[i].colors, clusters[j].colors)
			if distance < bestDistance || (distance == bestDistance && pairKeyLess(clusters[i], clusters[j], clusters[bestLeft], clusters[bestRight])) {
				bestDistance = distance
				bestLeft = i
				bestRight = j
			}
		}
	}

	return bestLeft, bestRight
}

func pairKeyLess(leftA, rightA, leftB, rightB *bankCluster) bool {
	if leftA.key < leftB.key {
		return true
	}
	if leftA.key > leftB.key {
		return false
	}
	return rightA.key < rightB.key
}
