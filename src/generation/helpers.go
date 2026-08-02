package generation

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"gonum.org/v1/plot/plotter"
)

func pointToTile(point Point, zoom int) Tile {
	latitudeRadians := point.latitude * math.Pi / 180
	n := math.Pow(2.0, float64(zoom))
	x := int((point.longitude + 180) / 360 * n)
	y := int((1.0 - math.Log(math.Tan(latitudeRadians)+1/math.Cos(latitudeRadians))/math.Pi) / 2.0 * n)
	return Tile{x: x, y: y}
}

func tileToPoint(tile Tile, zoom int) Point {
	n := math.Pow(2.0, float64(zoom))
	longitude := (float64(tile.x) / n * 360) - 180
	latitudeRadians := math.Atan(math.Sinh(math.Pi * (1 - 2*float64(tile.y)/n)))
	latitude := latitudeRadians * 180 / math.Pi
	return Point{latitude: latitude, longitude: longitude}
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func printProgress(max int, current int) {

	barLength := 20

	var buf strings.Builder

	buf.WriteString("\r[")
	progressPrev := float64(current-1) / float64(max)
	progress := float64(current) / float64(max)
	if int(progress*float64(barLength)+0.5) == int(progressPrev*float64(barLength)+0.5) && current != max {
		return
	}
	for range int(progress*float64(barLength) + 0.5) { // + 0.5 for correct rounding
		buf.WriteString("=")
	}
	for range int(float64(1-progress)*float64(barLength) + 0.5) {
		buf.WriteString(" ")
	}
	fmt.Fprintf(&buf, "] %5.2f%% - %v/%v", progress*100, current, max)
	fmt.Print(buf.String())
}

// gets all the tiles for a given zoom level with data points in them
func getTilesWithData(points []Point, zoom int) map[Tile]bool {
	tileSet := createTileSet()

	for _, point := range points {
		tile := pointToTile(point, zoom)
		tileSet[tile] = true
	}
	return tileSet
}

// converts a list of points to plotter.XYs
func pointListToPlotterXY(route []Point) plotter.XYs {
	pts := make(plotter.XYs, len(route))
	for i := range pts {
		pts[i].X = route[i].longitude
		pts[i].Y = route[i].latitude
	}
	return pts
}

func tempPathToKey(tempPath, prefix string) string {
	split := strings.Split(tempPath, "/")
	return filepath.Join(prefix, strings.Join(split[len(split)-3:], "/"))
}
