package actionprocessors

import (
	"strings"

	"github.com/disintegration/imaging"
)

// ResampleFilterByString returns filter value
func ResampleFilterByString(filter string, def imaging.ResampleFilter) imaging.ResampleFilter {
	switch strings.ToLower(filter) {
	case "nearest_neighbor", "nn":
		return imaging.NearestNeighbor
	case "box":
		return imaging.Box
	case "linear":
		return imaging.Linear
	case "hermite":
		return imaging.Hermite
	case "mitchell_netravali", "mn":
		return imaging.MitchellNetravali
	case "catmull_rom", "cr":
		return imaging.CatmullRom
	case "bspline", "bs":
		return imaging.BSpline
	case "gaussian":
		return imaging.Gaussian
	case "bartlett":
		return imaging.Bartlett
	case "lanczos":
		return imaging.Lanczos
	case "hann":
		return imaging.Hann
	case "hamming":
		return imaging.Hamming
	case "blackman":
		return imaging.Blackman
	case "welch":
		return imaging.Welch
	case "cosine":
		return imaging.Cosine
	}
	return def
}
