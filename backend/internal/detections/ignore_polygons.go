package detections

import "github.com/rtspanda/rtspanda/internal/cameras"

func shouldIgnoreByPolygon(polygons [][]cameras.Point, detection Detection, frameWidth int, frameHeight int) bool {
	if len(polygons) == 0 || frameWidth <= 0 || frameHeight <= 0 {
		return false
	}

	cx := detection.BBox.X + (detection.BBox.Width / 2)
	cy := detection.BBox.Y + (detection.BBox.Height / 2)

	x := float64(cx) / float64(frameWidth)
	y := float64(cy) / float64(frameHeight)

	if x < 0 || x > 1 || y < 0 || y > 1 {
		return false
	}

	for _, polygon := range polygons {
		if pointInPolygonNormalized(x, y, polygon) {
			return true
		}
	}
	return false
}

// Ray-casting point-in-polygon test in normalized [0..1] coordinates.
func pointInPolygonNormalized(x, y float64, polygon []cameras.Point) bool {
	n := len(polygon)
	if n < 3 {
		return false
	}

	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi := polygon[i].X
		yi := polygon[i].Y
		xj := polygon[j].X
		yj := polygon[j].Y

		intersects := (yi > y) != (yj > y)
		if intersects {
			den := yj - yi
			if den != 0 {
				xCross := (xj-xi)*(y-yi)/den + xi
				if x < xCross {
					inside = !inside
				}
			}
		}
		j = i
	}
	return inside
}
