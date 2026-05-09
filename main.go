package main

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"modular_crawler/parsing"
	"os"
	"runtime"
	"slices"
	"strconv"

	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg/draw"
	"gonum.org/v1/plot/vg/vgimg"

	"github.com/joho/godotenv"
)

var inputDirectory string = "C:/Users/Gregyr/Desktop/DESKTOP/CODE/py/actually useful/scraping/Coros Activities/gpx_data/run/"
var rootDir string = "C:/Users/Gregyr/Desktop/DESKTOP/CODE/py/actually useful/scraping/Coros Activities/gpx_data/"

var onlyRuns = false
var outputDirectory string = "tiles/"

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

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	inputDirectoryEnv, inputDirOk := os.LookupEnv("RUN_DIRECTORY")
	rootDirEnv, rootDirOk := os.LookupEnv("GPX_DIRECTORY")

	if !inputDirOk || !rootDirOk {
		log.Fatal("Missing Environment variables: RUN_DIRECTORY or GPX_DIRECTORY")
	} else {
		inputDirectory = inputDirectoryEnv
		rootDir = rootDirEnv
	}

	colorEnv, colorOk := os.LookupEnv("COLOR")

	if _, ok := colorSchemes[colorEnv]; colorOk && ok {
		colorScheme = colorSchemes[colorEnv]
	} else {
		var color string
		for name := range colorSchemes {
			fmt.Printf("Color \"%s\"\n", name)
		}
		fmt.Print("Choose Color:")
		fmt.Scanln(&color)
		setColorScheme(color)
	}

	runsEnv, runsOk := os.LookupEnv("ONLY_RUNS")
	if runsOk && runsEnv == "true" {
		onlyRuns = true
	} else if runsOk && runsEnv == "false" {
		onlyRuns = false
	} else {
		var runs string
		fmt.Println("Only Map Running activities? (Leave empty for no, input anything for yes)")
		fmt.Scanln(&runs)
		if len(runs) > 0 {
			onlyRuns = true
		}
	}
	generateTileImages()
}

func generateTileImages() {

	// create outputDirectory if it does not exist
	outDirExists, _ := exists(outputDirectory)
	if !outDirExists {
		os.Mkdir(outputDirectory, os.ModePerm)
	}

	points, routes := extractAllPointsAndRoutes()

	numWorkers := runtime.NumCPU()
	bufferSize := 100
	pool := NewWorkerPool(numWorkers, bufferSize)
	pool.Start()

	jobCount := 0
	for _, zoom := range zoomLevels {
		tiles := getTilesWithData(points, zoom)
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
// returns a list of all points aswell as a list of routes
func extractAllPointsAndRoutes() ([]Point, []Route) {

	entries := []string{}
	if !onlyRuns {
		folders, err := os.ReadDir(rootDir)
		if err != nil {
			log.Fatal(err)
		}
		for _, f := range folders {
			if f.Name() != "other" {
				e, err := os.ReadDir(rootDir + f.Name())
				if err != nil {
					log.Fatal(err)
				}
				for _, en := range e {
					entries = append(entries, f.Name()+"/"+en.Name())
				}
			}
		}

	} else {
		e, err := os.ReadDir(inputDirectory)
		if err != nil {
			log.Fatal(err)
		}
		for _, en := range e {
			entries = append(entries, en.Name())
		}
	}
	points := []Point{} // all points
	routes := []Route{} // all points by route

	for i, e := range entries {
		log.Println("Extracting Route", i, e)
		route, err := getRouteFromEntryString(e)
		if err != nil {
			log.Println(err)
		}
		points = slices.Concat(points, route.points)
		routes = append(routes, route)
	}
	return points, routes
}

// extracts the route from an entry
// the entry is just the entry itself and does not contain the path
func getRouteFromEntryString(entry string) (Route, error) {

	// load file
	var fileContent []byte
	if onlyRuns {
		var err error
		fileContent, err = os.ReadFile(inputDirectory + entry)
		if err != nil {
			return Route{}, err
		}
	} else {
		var err error
		fileContent, err = os.ReadFile(rootDir + entry)
		if err != nil {
			return Route{}, err
		}
	}
	// get Node Structure
	rootNode, err := parsing.ParseXML(string(fileContent))
	if err != nil {
		return Route{}, err
	}

	// extract nodes storing position info
	positionNodes := parsing.EvaluateXPath(rootNode, "/gpx/trk/trkseg/trkpt")
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

func routeToPlotterXY(route []Point) plotter.XYs {
	pts := make(plotter.XYs, len(route))
	for i := range pts {
		pts[i].X = route[i].longitude
		pts[i].Y = route[i].latitude
	}
	return pts
}

func plotRoutes(routes []Route, p1 Point, p2 Point, tile Tile, zoom int) {
	p := plot.New()

	for _, route := range routes { // check wheter a route even intersects the tile, plotting can be skipped otherwise
		if !(route.max.latitude < p2.latitude ||
			route.max.longitude < p1.longitude ||
			route.min.latitude > p1.latitude ||
			route.min.longitude > p2.longitude) {
			line, err := plotter.NewLine(routeToPlotterXY(route.points))
			if err != nil {
				log.Fatal(err)
			}

			line.LineStyle.Color = color.RGBA{R: 50, G: 50, B: 50, A: 50}
			line.LineStyle.Width = 1
			line.StepStyle = plotter.NoStep
			p.Add(line)
		}
	}

	p.HideAxes()
	p.Title.Padding = 0

	plotAccPadding := 0.2
	tileWidthDegrees := 360.0 / math.Pow(2.0, float64(zoom))
	epsilon := tileWidthDegrees * plotAccPadding / 256

	p.X.Max = p2.longitude + epsilon
	p.X.Min = p1.longitude - epsilon
	p.Y.Max = p1.latitude + epsilon
	p.Y.Min = p2.latitude - epsilon
	p.X.Padding = -0.2
	p.Y.Padding = -0.2 // plotAccPadding idk how to make it vg.length
	p.BackgroundColor = color.Transparent

	c := vgimg.PngCanvas{Canvas: vgimg.NewWith(
		vgimg.UseWH(256, 256),
		vgimg.UseBackgroundColor(color.Transparent),
	)}
	p.Draw(draw.New(c))
	outPath := fmt.Sprintf("%s/%v/%v/", outputDirectory, zoom, tile.x)
	os.MkdirAll(outPath, os.ModePerm)

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

func setColorScheme(color string) {
	cs, ok := colorSchemes[color]
	if ok {
		outputDirectory = fmt.Sprintf("tiles_%s", color)
		colorScheme = cs
	}
}

func printProgress(max int, current int) {

	barLength := 20

	fmt.Print("\r[")
	progress := float64(current) / float64(max)
	for range int(progress * float64(barLength)) {
		fmt.Print("=")
	}
	for range int(float64(1-progress) * float64(barLength)) {
		fmt.Print(" ")
	}
	fmt.Printf("] %5.2f%%", progress*100)
}
