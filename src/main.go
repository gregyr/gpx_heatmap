package main

import "github.com/gregyr/gpx_heatmap/src/generation"

func main() {
	// temporary call
	generation.SetupEnvironment()
	generation.Run(generation.LocalStore{RootDir: generation.OutputDirectory})
}
