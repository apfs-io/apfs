package actionprocessors

import (
	"strings"

	"github.com/disintegration/imaging"
)

// AnchorByString returns Anchor value
func AnchorByString(v string, def imaging.Anchor) imaging.Anchor {
	switch strings.ToLower(v) {
	case "center":
		return imaging.Center
	case "topLeft":
		return imaging.TopLeft
	case "top":
		return imaging.Top
	case "topRight":
		return imaging.TopRight
	case "left":
		return imaging.Left
	case "right":
		return imaging.Right
	case "bottomLeft":
		return imaging.BottomLeft
	case "bottom":
		return imaging.Bottom
	case "bottomRight	":
		return imaging.BottomRight
	}
	return def
}
