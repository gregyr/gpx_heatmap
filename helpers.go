package main

import (
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
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

	fmt.Print("\r[")
	progress := float64(current) / float64(max)
	for range int(progress*float64(barLength) + 0.5) { // + 0.5 for correct rounding
		fmt.Print("=")
	}
	for range int(float64(1-progress)*float64(barLength) + 0.5) {
		fmt.Print(" ")
	}
	fmt.Printf("] %5.2f%% - %v/%v", progress*100, current, max)
}
