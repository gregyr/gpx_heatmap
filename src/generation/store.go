package generation

import (
	"log"
	"os"
	"path/filepath"
)

type store interface {
	GetFileContent(string) ([]byte, error)
	WriteFileContent(string, []byte) error
}

type LocalStore struct {
	RootDir string
}

// reads the file at the given key relative to the store's root dir
func (s LocalStore) GetFileContent(key string) ([]byte, error) {
	// fullPath := filepath.Join(s.rootDir, key)
	return nil, nil
}

// writes the file at the given key relative to the store's root dir
func (s LocalStore) WriteFileContent(key string, data []byte) error {
	fullPath := filepath.Join(s.RootDir, key)
	os.MkdirAll(filepath.Dir(fullPath), os.ModePerm)
	log.Printf("Writing to %s\n", fullPath)
	return os.WriteFile(fullPath, data, os.ModePerm)
}
