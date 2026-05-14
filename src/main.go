package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/joho/godotenv"
)

var inputDirectory string = ""

var newGpxPath string = "newgpx.txt"

var ignoreDir string = ""
var recursiveSearch bool = false
var outputDirectory string = "tiles/"
var onlyNewTiles bool = false

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

var colorSchemes map[string]ColorScheme = map[string]ColorScheme{
	"red": {
		colorStart: color.RGBA{R: 223, G: 0, B: 64, A: 255},
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

func main() {
	setupEnvironment()
	generateTileImages()
}

func generateTileImages() {

	// create outputDirectory if it does not exist
	outDirExists, _ := exists(outputDirectory)
	if !outDirExists {
		os.Mkdir(outputDirectory, os.ModePerm)
	}

	points, routes, newPoints := extractAllPointsAndRoutes() // always load all points as it would be too annoying / not efficient to check if a route intersects a new route

	numWorkers := runtime.NumCPU()
	bufferSize := 100
	pool := NewWorkerPool(numWorkers, bufferSize)
	pool.Start()

	jobCount := 0
	for _, zoom := range zoomLevels {
		var tiles map[Tile]bool
		if onlyNewTiles {
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
				tile:   tile,
				zoom:   zoom,
				p1:     p1,
				p2:     p2,
				routes: routes,
			}
			pool.Submit(job)
			jobCount++
			tileCount++
			printProgress(len(tiles), tileCount)
		}
		fmt.Println("\033[?25h")
	}

	pool.Close()
	log.Printf("Processed %d tiles\n", jobCount)
}

// extracts all points and routes from all gpx files in the given inputDirectory
// a route is a list of points
// returns a list of all points aswell as a list of routes and a list of all new Points, which is empty if not needed
func extractAllPointsAndRoutes() ([]Point, []Route, []Point) {

	newGpxFileNames := []string{}
	if onlyNewTiles {
		content, err := os.ReadFile(newGpxPath)
		if err != nil {
			log.Fatal(err)
		}
		stringContent := string(content)
		newGpxFileNames = strings.Split(stringContent, "\n")
		for i, n := range newGpxFileNames {
			newGpxFileNames[i] = strings.Trim(n, " \n\r")
		}
	}

	entries := []string{} // MAKE THIS OPTIONALLY RECURSIVE OR IGNORE ENTRY
	if recursiveSearch {
		dirs := []string{} // store dir paths

		// dir Entries of input dir
		dirEntries, err := os.ReadDir(inputDirectory)
		if err != nil {
			log.Fatal(err)
		}

		// fill dirs with the dirs from input dir
		for _, e := range dirEntries {
			if e.IsDir() {
				dirs = append(dirs, e.Name())
			} else {
				entries = append(entries, e.Name())
			}
		}

		// pop element from dir list as long as there are elements
		for len(dirs) > 0 {
			dir := dirs[0]
			dirs = dirs[1:]
			if ignoreDir != "" && strings.Contains(dir, ignoreDir) { // check if valid dir name else ignore
				continue
			}
			entrs, err := os.ReadDir(inputDirectory + dir) // get all entries
			if err != nil {
				log.Fatal(err)
			}
			for _, e := range entrs { // add entries recursively to dirs or entries depending on filetype
				if e.IsDir() {
					dirs = append(dirs, dir+"/"+e.Name()) // name relative to input directory
				} else {
					entries = append(entries, dir+"/"+e.Name())
				}
			}
		}

	} else {
		e, err := os.ReadDir(inputDirectory)
		if err != nil {
			log.Fatal(err)
		}
		for _, en := range e {
			if !en.IsDir() {
				entries = append(entries, en.Name())
			}
		}
	}
	points := []Point{} // all points
	routes := []Route{} // all points by route
	newPoints := []Point{}

	for i, e := range entries {
		log.Println("Extracting Route", i, e)
		route, err := getRouteFromEntryString(e)
		if err != nil {
			log.Println(err)
		}
		points = slices.Concat(points, route.points)
		routes = append(routes, route)
		if onlyNewTiles && slices.Contains(newGpxFileNames, strings.Split(e, "/")[len(strings.Split(e, "/"))-1]) {
			newPoints = slices.Concat(newPoints, route.points)
		}
	}
	return points, routes, newPoints
}

// extracts the route from an entry
// the entry is just the entry itself and does not contain the path
func getRouteFromEntryString(entry string) (Route, error) {

	// load file
	fileContent, err := os.ReadFile(inputDirectory + entry)
	if err != nil {
		return Route{}, err
	}
	// get Node Structure
	rootNode, err := ParseXML(string(fileContent))
	if err != nil {
		return Route{}, err
	}

	// extract nodes storing position info
	positionNodes := EvaluateXPath(rootNode, "/gpx/trk/trkseg/trkpt")
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

// plots a route respective on a tile at a given zoom level
// p1 and p2 are the tiles XY and X+1Y+1 coordinates
func plotRoutes(routes []Route, p1 Point, p2 Point, tile Tile, zoom int) {
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
	outPath := fmt.Sprintf("%s/%v/%v/", outputDirectory, zoom, tile.x)
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

	// output file
	f, err := os.Create(fmt.Sprintf("%s%v.png", outPath, tile.y))
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	_, err = c.WriteTo(f)
	if err != nil {
		log.Fatal(err)
	}

	err = f.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// sets the colorscheme based on a given string, handles wrong strings
func setColorScheme(color string) {
	cs, ok := colorSchemes[color]
	if ok {
		outputDirectory = fmt.Sprintf("%s_%s", outputDirectory, color)
		colorScheme = cs
	}
}

// loads environment variables from .env
// local config for missing optional Vars
func setupEnvironment() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	inputDirectoryEnv, inputDirOk := os.LookupEnv("GPX_DIRECTORY")

	if !inputDirOk {
		log.Fatal("Missing Environment: GPX_DIRECTORY")
	} else {
		inputDirectory = inputDirectoryEnv
	}

	outputEnv, outputOk := os.LookupEnv("OUTPUT")
	if outputOk && len(outputEnv) > 0 {
		outputDirectory = outputEnv
	}

	colorEnv, colorOk := os.LookupEnv("COLOR")

	if colorOk {
		setColorScheme(colorEnv)
	} else {
		log.Println("Missing optional env 'COLOR', choose Color")
		var color string
		for name := range colorSchemes {
			fmt.Printf("Color \"%s\"\n", name)
		}
		fmt.Print("Choose Color:")
		fmt.Scanln(&color)
		setColorScheme(color)
	}

	recursiveEnv, recursiveOk := os.LookupEnv("RECURSIVE")
	if recursiveOk && recursiveEnv == "true" {
		recursiveSearch = true
	} else if recursiveOk && recursiveEnv == "false" {
		recursiveSearch = false
	} else {
		log.Println("Missing optional env 'RECURSIVE', choose:")
		var runs string
		fmt.Println("Search input dir recursively? (empty input for no, input anything for yes)")
		fmt.Scanln(&runs)
		if len(runs) > 0 {
			recursiveSearch = true
		}
	}

	ignoreDirEnv, ignoreDirOk := os.LookupEnv("IGNORE_DIR")
	if ignoreDirOk {
		ignoreDir = ignoreDirEnv
	}

	onlyNewTilesEnv, onlyNewTilesOk := os.LookupEnv("ONLY_NEW")
	if onlyNewTilesOk && onlyNewTilesEnv == "true" {
		onlyNewTiles = true
	} else if recursiveOk && recursiveEnv != "false" {
		log.Println("Not a valid input for Environment Variable 'ONLY_NEW', expects: true/false, using false")
	}

	newGpxPathEnv, newGpxPathOk := os.LookupEnv("NEW_GPX_PATH")
	if newGpxPathOk {
		newGpxPath = newGpxPathEnv
	}

}
