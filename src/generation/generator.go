package generation

import (
	"fmt"
	"image/color"
	"io"
	"log"
	"math"
	"os"
	"runtime"
	"slices"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/gregyr/gpx_heatmap/src/xml"
)

var zoomLevels = []int{5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

type Point struct {
	latitude  float64
	longitude float64
}

type Tile struct {
	x int
	y int
}

type Route struct {
	points []Point
	min    Point
	max    Point
}

type ColorScheme struct {
	colorStart color.RGBA
	colorEnd   color.RGBA
}

type RunConfig struct {
	OnlyNew          bool
	Prefix           string
	Color            string
	stagingDirectory string
}

var colorSchemes map[string]ColorScheme = map[string]ColorScheme{
	"red": {
		colorStart: color.RGBA{R: 223, G: 0, B: 64, A: 100},
		colorEnd:   color.RGBA{R: 255, G: 192, B: 146, A: 255}},
	"blue": {
		colorStart: color.RGBA{R: 0, G: 64, B: 111, A: 255},
		colorEnd:   color.RGBA{R: 195, G: 220, B: 255, A: 255}},
	"blue-red": {
		colorStart: color.RGBA{R: 0, G: 64, B: 111, A: 255},
		colorEnd:   color.RGBA{R: 255, G: 192, B: 146, A: 255}},
	"red-blue": {
		colorStart: color.RGBA{R: 223, G: 0, B: 64, A: 255},
		colorEnd:   color.RGBA{R: 195, G: 220, B: 255, A: 255}},
}

var colorScheme = colorSchemes["red"]

func Run(tileStore store, gpxKeyStore gpxKeyStore, gpxFileStore store, config RunConfig) error {

	// create staging directory
	tempDir, err := os.MkdirTemp("", "tiles-*")
	if err != nil {
		fmt.Printf("Failed to initialize staging director: %v", err)
		os.Exit(1)
	}
	config.stagingDirectory = tempDir
	stagedFiles := NewSafeStringSet()

	points, routes, newPoints, err := extractAllPointsAndRoutes(gpxKeyStore, gpxFileStore, config) // always load all points as it would be too annoying / not efficient to check if a route intersects a new route
	if err != nil {
		return err
	}

	numWorkers := runtime.NumCPU()
	bufferSize := 100
	pool := NewWorkerPool(numWorkers, bufferSize)
	pool.Start()

	jobCount := 0
	for _, zoom := range zoomLevels {
		var tiles map[Tile]bool
		if config.OnlyNew {
			tiles = getTilesWithData(newPoints, zoom) // only get Tiles that intersect the new points
		} else {
			tiles = getTilesWithData(points, zoom)
		}
		log.Println("Generating zoom", zoom)
		fmt.Print("\033[?25l")
		tileCount := 0
		for tile := range tiles {
			p1 := tileToPoint(tile, zoom)
			p2 := tileToPoint(Tile{x: tile.x + 1, y: tile.y + 1}, zoom)

			job := PlotJob{
				tile:        tile,
				zoom:        zoom,
				p1:          p1,
				p2:          p2,
				routes:      routes,
				stagedFiles: stagedFiles,
				config:      config,
			}
			pool.Submit(job)
			jobCount++
			tileCount++
			printProgress(len(tiles), tileCount)
		}
		fmt.Println("\033[?25h")
	}

	pool.Close()

	progress := 0
	log.Println("Writing Final Files to Store")
	for path, f := range stagedFiles.values {
		defer f.Close()
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			log.Printf("seek failed: %v", err)
			continue
		}

		data, err := io.ReadAll(f)
		if err != nil {
			log.Printf("read failed: %v", err)
			continue
		}

		tileStore.WriteFileContent(tempPathToKey(path, config.Prefix), data)
		progress++
		printProgress(len(stagedFiles.values), progress)
	}
	fmt.Println("\033[?25h")
	log.Printf("Processed %d tiles\n", jobCount)
	os.RemoveAll(tempDir) //cleanup
	return nil
}

// extracts all points and routes from all gpx files in the given inputDirectory
// a route is a list of points
// returns a list of all points aswell as a list of routes and a list of all new Points, which is empty if not needed
func extractAllPointsAndRoutes(gpxKeyStore gpxKeyStore, gpxFileStore store, config RunConfig) ([]Point, []Route, []Point, error) {

	newGpxFileNames := []string{}
	if config.OnlyNew {
		var err error
		newGpxFileNames, err = gpxKeyStore.GetNewActivityFileKeys()
		if err != nil {
			return []Point{}, []Route{}, []Point{}, err
		}
	}

	entries, err := gpxKeyStore.GetActivityFileKeys()
	if err != nil {
		return []Point{}, []Route{}, []Point{}, err
	}

	points := []Point{} // all points
	routes := []Route{} // all points by route
	newPoints := []Point{}

	for i, e := range entries {
		log.Println("Extracting Route", i, e)
		route, err := getRouteFromEntryString(e, gpxFileStore)
		if err != nil {
			log.Println(err)
		}
		points = slices.Concat(points, route.points)
		routes = append(routes, route)
		if config.OnlyNew && slices.Contains(newGpxFileNames, e) {
			newPoints = slices.Concat(newPoints, route.points)
		}
	}
	return points, routes, newPoints, nil
}

// extracts the route from an entry
// the entry is just the entry itself and does not contain the path
func getRouteFromEntryString(entry string, gpxStore store) (Route, error) {

	// load file
	fileContent, err := gpxStore.GetFileContent(entry)
	if err != nil {
		return Route{}, err
	}
	// get Node Structure
	rootNode, err := xml.ParseXML(string(fileContent))
	if err != nil {
		return Route{}, err
	}

	// extract nodes storing position info
	positionNodes := xml.EvaluateXPath(rootNode, "/gpx/trk/trkseg/trkpt")
	route := []Point{}
	// parse the node attributes storing the lat and lon info
	minLat := math.Inf(1)
	minLon := math.Inf(1)
	maxLat := math.Inf(-1)
	maxLon := math.Inf(-1)
	for _, node := range positionNodes {
		lat, err := strconv.ParseFloat(node.Attributes["lat"], 64)
		if err != nil {
			continue
		}
		maxLat = max(maxLat, lat)
		minLat = min(minLat, lat)
		lon, err := strconv.ParseFloat(node.Attributes["lon"], 64)
		if err != nil {
			continue
		}
		maxLon = max(maxLon, lon)
		minLon = min(minLon, lon)
		route = append(route, Point{latitude: lat, longitude: lon})
	}
	return Route{points: route, max: Point{latitude: maxLat, longitude: maxLon}, min: Point{latitude: minLat, longitude: minLon}}, nil
}

// plots a route respective on a tile at a given zoom level
// p1 and p2 are the tiles XY and X+1Y+1 coordinates
func plotRoutes(routes []Route, p1 Point, p2 Point, tile Tile, zoom int, stagedFiles *SafeStringSet, config RunConfig) {
	p := plot.New()

	var routeBrightness uint8 = 50

	for _, route := range routes { // check wheter a route even intersects the tile, plotting can be skipped otherwise
		if !(route.max.latitude < p2.latitude ||
			route.max.longitude < p1.longitude ||
			route.min.latitude > p1.latitude ||
			route.min.longitude > p2.longitude) {
			line, err := plotter.NewLine(pointListToPlotterXY(route.points))
			if err != nil {
				log.Fatal(err)
			}

			line.LineStyle.Color = color.RGBA{R: routeBrightness, G: routeBrightness, B: routeBrightness, A: routeBrightness}
			line.LineStyle.Width = 1
			line.StepStyle = plotter.NoStep
			p.Add(line)
		}
	}

	// set plot options
	p.HideAxes()
	p.Title.Padding = 0

	// make plot slightly larger for less color deviance at the border
	plotAccPadding := 0.2 // 0.2 to both limit stretching and maximize padding
	tileWidthDegrees := 360.0 / math.Pow(2.0, float64(zoom))
	epsilon := tileWidthDegrees * plotAccPadding / 256

	p.X.Max = p2.longitude + epsilon
	p.X.Min = p1.longitude - epsilon
	p.Y.Max = p1.latitude + epsilon
	p.Y.Min = p2.latitude - epsilon
	p.X.Padding = -0.2
	p.Y.Padding = -0.2 // plotAccPadding idk how to make it vg.length
	p.BackgroundColor = color.Transparent

	// create transparent canvas
	c := vgimg.PngCanvas{Canvas: vgimg.NewWith(
		vgimg.UseWH(256, 256),
		vgimg.UseBackgroundColor(color.Transparent),
	)}
	p.Draw(draw.New(c))

	// format output
	outPath := fmt.Sprintf("%s/%v/%v/", config.stagingDirectory, zoom, tile.x)
	os.MkdirAll(outPath, os.ModePerm)

	// color pixels based on their alpha value

	imageBounds := c.Image().Bounds()
	for x := range imageBounds.Dx() {
		for y := range imageBounds.Dy() {
			pxlColor := c.Image().At(x, y)
			_, _, _, a := pxlColor.RGBA()

			if a != 0 {

				lightness := float64(a) / 65535.0
				r := uint8(float64(colorScheme.colorStart.R)*(1-lightness) + float64(colorScheme.colorEnd.R)*lightness)
				g := uint8(float64(colorScheme.colorStart.G)*(1-lightness) + float64(colorScheme.colorEnd.G)*lightness)
				b := uint8(float64(colorScheme.colorStart.B)*(1-lightness) + float64(colorScheme.colorEnd.B)*lightness)
				c.Image().Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}

	outputFile := fmt.Sprintf("%s%v.png", outPath, tile.y)
	// output file
	f, err := os.Create(outputFile)
	if err != nil {
		log.Fatal(err)
	}

	_, err = c.WriteTo(f)
	if err != nil {
		log.Fatal(err)
	}

	if stagedFiles != nil {
		stagedFiles.Add(outputFile, f)
	}
}

// sets the colorscheme based on a given string, handles wrong strings
func ValidColorScheme(color string) bool {
	_, ok := colorSchemes[color]
	return ok
}
