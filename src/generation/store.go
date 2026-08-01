package generation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type store interface {
	GetFileContent(string) ([]byte, error)
	//GetAllFileKeys() ([]byte, error)
	WriteFileContent(string, []byte) error
}

type ActivityType string

const (
	typeRun         ActivityType = "run"
	typeBike        ActivityType = "bike"
	typeHike        ActivityType = "hike"
	typeSki         ActivityType = "ski"
	typeOther       ActivityType = "other"
	typeAllExplicit ActivityType = "all"
	typeAll         ActivityType = "" // leaving the type in the story empty defaults to all
)

type gpxKeyStore interface {
	GetActivityFileKeys() ([]string, error)
	GetNewActivityFileKeys() ([]string, error)
}

type LocalGpxKeyStore struct {
	Type       ActivityType
	RootDir    string
	NewGpxFile string
}

type LocalStore struct {
	RootDir string
}

// reads the file at the given key relative to the store's root dir
func (s LocalStore) GetFileContent(key string) ([]byte, error) {
	fullPath := filepath.Join(s.RootDir, key)
	return os.ReadFile(fullPath)
}

// writes the file at the given key relative to the store's root dir
func (s LocalStore) WriteFileContent(key string, data []byte) error {
	fullPath := filepath.Join(s.RootDir, key)
	os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
	return os.WriteFile(fullPath, data, os.ModePerm)
}

func (p LocalGpxKeyStore) GetActivityFileKeys() ([]string, error) {
	entries := []string{}
	if p.Type == typeAll || p.Type == typeAllExplicit {
		dirs := []string{} // store dir paths

		// dir Entries of input dir
		dirEntries, err := os.ReadDir(p.RootDir)
		if err != nil {
			return []string{}, fmt.Errorf("reading store root directory entries: %v", err)
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
			if dir == string(typeOther) { // ignore "other" dir
				continue
			}
			entrs, err := os.ReadDir(p.RootDir + dir) // get all entries
			if err != nil {
				return []string{}, fmt.Errorf("reading activity directory entries: %v", err)
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
		e, err := os.ReadDir(filepath.Join(p.RootDir, string(p.Type)))
		if err != nil {
			return []string{}, fmt.Errorf("reading activity directory: %v", err)
		}
		for _, en := range e {
			if !en.IsDir() {
				entries = append(entries, filepath.Join(string(p.Type), en.Name()))
			}
		}
	}
	return entries, nil
}

func (p LocalGpxKeyStore) GetNewActivityFileKeys() ([]string, error) {
	newGpxFileNames := []string{}
	content, err := os.ReadFile(p.NewGpxFile)
	if err != nil {
		return []string{}, fmt.Errorf("reading new gpx files")
	}
	stringContent := string(content)
	newGpxFileNames = strings.Split(stringContent, "\n")
	for i, n := range newGpxFileNames {
		newGpxFileNames[i] = strings.Trim(n, " \n\r")
	}
	return newGpxFileNames, nil
}
