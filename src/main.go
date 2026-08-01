package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gregyr/gpx_heatmap/src/generation"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	environment := os.Getenv("ENVIRONMENT")

	log.Printf("Running with Environment: %s\n", environment)

	var runError error
	switch environment {
	case "local":
		runError = generation.Run(SetupLocal())
	default:
		runError = fmt.Errorf("Missing or Invalid Environment\nProvided: %s\nValid Environments: \"local\"", environment)
	}

	if runError != nil {
		log.Fatalf("Failed to Generate Tiles: \n%v\n", runError)
	}
	log.Println("Finished Successfully!")
}

func SetupLocal() (generation.LocalStore, generation.LocalGpxKeyStore, generation.LocalStore, generation.RunConfig) {

	var store generation.LocalStore
	var gpxKeyStore generation.LocalGpxKeyStore
	var gpxFileStore generation.LocalStore
	var config generation.RunConfig

	inputDirectoryEnv, inputDirOk := os.LookupEnv("GPX_DIRECTORY")
	if !inputDirOk {
		log.Fatal("Missing Environment: GPX_DIRECTORY")
	} else {
		gpxKeyStore.RootDir = inputDirectoryEnv
		gpxFileStore.RootDir = inputDirectoryEnv
	}

	newGpxPathEnv, newGpxPathOk := os.LookupEnv("NEW_GPX_PATH")
	if newGpxPathOk {
		gpxKeyStore.NewGpxFile = newGpxPathEnv
	}

	// empty means all so this is fine
	gpxKeyStore.Type = generation.ActivityType(os.Getenv("ACTIVITY_TYPE"))

	outputEnv, outputOk := os.LookupEnv("OUTPUT_ROOT")
	if outputOk && len(outputEnv) > 0 {
		store.RootDir = outputEnv
	} else {
		log.Fatal("Missing Environment: OUTPUT_ROOT")
	}
	colorEnv, colorOk := os.LookupEnv("COLOR")

	if colorOk && generation.ValidColorScheme(colorEnv) {
		config.Color = colorEnv
	} else {
		config.Color = "red"
	}

	onlyNewTilesEnv, onlyNewTilesOk := os.LookupEnv("ONLY_NEW")
	if onlyNewTilesOk && onlyNewTilesEnv == "true" {
		config.OnlyNew = true
	} else {
		config.OnlyNew = false
	}

	prefixEnv, prefixOk := os.LookupEnv("OUTPUT_PREFIX")
	if !prefixOk {
		config.Prefix = "tiles"
	} else {
		config.Prefix = prefixEnv
	}

	return store, gpxKeyStore, gpxFileStore, config
}
