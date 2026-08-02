## GPX Heatmap Generator

Generates a heatmap from `.gpx` files. It creates a Tilemap to be overlayed on any other type of map. An example is in the `index.html` file.

### Example Screenshot
![example heatmap](image.png "example heatmap")

## Usage

Create a `.env`, like `.env.example` shows <br>
Run `go build .` in the `src/` directory to build the application and run it **or** <br>
run `go run .` in the `src/` directory to build and run the application

## Environments

The Plan is to support different environments such as GCP Hosted, local, etc. Currently only "local" is a valid environment, configuration can be found in the Environment Variables Section for the "local" Environment

## Stores
This is where the `.gpx` files, tiles and data about new activites will be stored.
### Gpx Store
The Gpx Store is where the `.gpx` Files live. It can be a bucket, local directory, etc. It should have the following structure:
```
root
 ├── run
 ├── bike
 ├── hike
 ├── ski
 └── other
```
Directories may be omitted or added, a directory may only hold `.gpx` files. If you have for example a swimming directory, with only swimming `.gpx` files it can be added. The `other` directory will be ignored when loading all files.

### Gpx Key Store
The Gpx Key Store provides the Keys of Gpx Files relative to the Gpx Store's root, this is seperated, so that for different setups Gpx Key's may be provided through a database or similar.

### Tile Store
The Tile store is where the tiles will be stored, this again can be a bucket, local directory or similar
```
tiles
 ├── A
 └── B
```
### Local Setup
For the `"local"` Setup, you should provide a directory in the form of a Gpx Store and a Tile Store, the Gpx Key Store provides the Keys from the Gpx Store. The path's to these have to be provided through [environment variables](#local-setup)

### Local Database Setup
For the `"local_db"` Setup, you should provide a directory in the form of a Gpx Store and a Tile Store, the Gpx Key Database has to be provided through a file and has to contain a file_key, date and type column. The date column has to have activity dates in the format `"2006-01-02"` and the type needs to match the Gpx Store's type directories. The path's to these have to be provided through [environment variables](#local-setup)

### Environment variables

#### Config Variables
- `ENVIRONMENT` **mandatory** configures what setup is used currently supported: "local"
- `OUTPUT_PREFIX` **optional** configures the prefix used for storing the tiles, if left empty it defaults to `tiles`, if an empty string is provided, it will be placed in the root directory of your store
- `ONLY_NEW` **optional** whether to only generate new tiles based on new activity files, if true, the store specific way of providing the data of which tiles are new needs to be configured
- `COLOR` **optional** configures the color of the heatmap, defaults to `red`, currently supported: `red`, `blue`, `blue-red`, `red-blue`
- `ACTIVITY_TYPE` **optional** which activity types should be used from the store, defaults to `all`, currently excplicitly supported: `all`, `run`, `bike`, `hike`, `ski`. You can provide other types if they follow the structure described in the [Gpx Store](#gpx-store)

#### `"local"`-Setup variables

- `GPX_DIRECTORY` **mandatory**, configures the path to your [Gpx Store](#gpx-store) Directory
- `OUTPUT_ROOT` **mandatory**, configures the path to your [Tile Store](#tile-store) Directory
- `NEW_GPX_PATH` **optional**, if `ONLY_NEW` is enabled, this is how you provide the data for which activities are new, the format can be found in the newgpx.txt.example file

#### `"local_db"`-Setup variables

- `GPX_DIRECTORY` **mandatory**, configures the path to your [Gpx Store](#gpx-store) Directory
- `OUTPUT_ROOT` **mandatory**, configures the path to your [Tile Store](#tile-store) Directory
- `GPX_DATABASE` **mandatory**, configures the path to your [Gpx Key Database](#gpx-key-store) File
- `AFTER_DATE` **optional**, sets the date after which activities are classified as new, if not provided, every activity will be classified as new

### What's open

- Add a `"gcp"` environment
    - Gpx Store as a Bucket
    - Gpx Key Store as Database -> New Gpx calculated based on last run
    - Tile Store as a Bucket
- Add support for using previously calculated bounds for every file to optimise file loading -> probably through the above mentioned interface
- Improve xml parser