package media

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// Store defines the interface for saving, retrieving, and deleting media assets
type Store interface {
	// Save stores data from reader to a specific relative path within a subdirectory
	// returns the final relative path used (might include generated filename) and error
	Save(assetType AssetType, relativeDirHint string, filenameHint string, data io.Reader) (string, error)
	// Get retrieves a reader for an asset
	Get(relativePath string) (io.ReadCloser, os.FileInfo, error)
	// Delete removes an asset
	Delete(relativePath string) error
	// GetFullPath returns the absolute filesystem path for a relative asset path
	GetFullPath(relativePath string) (string, error)
	// EnsureDir makes sure a specific asset type directory exists
	EnsureDir(assetType AssetType) (string, error)
}

// LocalStorage implements the Store interface using the local filesystem
type LocalStorage struct {
	basePath        string               // absolute path to the MEDIA_STORAGE_PATH
	subDirMap       map[AssetType]string // maps AssetType to subdirectory name (e.g., "thumbnails")
	resolvedPathMap map[AssetType]string // maps AssetType to full absolute path
}

// NewLocalStorage creates a new local filesystem store
func NewLocalStorage(basePath string, subDirs map[AssetType]string) (*LocalStorage, error) {
	absBasePath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid base storage path '%s': %w", basePath, err)
	}

	if err := os.MkdirAll(absBasePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base storage directory '%s': %w", absBasePath, err)
	}

	resolvedPaths := make(map[AssetType]string)
	for assetType, subDir := range subDirs {
		fullPath := filepath.Join(absBasePath, subDir)
		if !strings.HasPrefix(filepath.Clean(fullPath), absBasePath) {
			return nil, fmt.Errorf("invalid subdirectory configuration: '%s' resolves outside base path '%s'", subDir, absBasePath)
		}
		resolvedPaths[assetType] = fullPath
	}

	log.Printf("media.store: Initialized LocalStorage at %s", absBasePath)
	return &LocalStorage{
		basePath:        absBasePath,
		subDirMap:       subDirs,
		resolvedPathMap: resolvedPaths,
	}, nil
}

// getAssetTypeDir resolves the absolute path for a given asset type
func (ls *LocalStorage) getAssetTypeDir(assetType AssetType) (string, error) {
	dirPath, ok := ls.resolvedPathMap[assetType]
	if !ok {
		log.Printf("media.store: Warning - Asset type '%s' not explicitly configured, using as subdirectory name", assetType)
		dirPath = filepath.Join(ls.basePath, string(assetType))

		if !strings.HasPrefix(filepath.Clean(dirPath), ls.basePath) {
			return "", fmt.Errorf("asset type '%s' resolves outside base path", assetType)
		}
		ls.resolvedPathMap[assetType] = dirPath
	}
	return dirPath, nil
}

// EnsureDir creates the directory for the asset type if it doesn't exist
func (ls *LocalStorage) EnsureDir(assetType AssetType) (string, error) {
	dirPath, err := ls.getAssetTypeDir(assetType)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to ensure directory '%s': %w", dirPath, err)
	}
	return dirPath, nil
}

// Save data to the store. filenameHint can be empty to generate one (e.g., UUID)
// relativeDirHint allows for further structure within the asset type's main dir (e.g., album ID)
func (ls *LocalStorage) Save(assetType AssetType, relativeDirHint string, filenameHint string, data io.Reader) (string, error) {
	baseAssetDir, err := ls.EnsureDir(assetType)
	if err != nil {
		return "", err
	}

	targetDir := baseAssetDir
	if relativeDirHint != "" {
		targetDir = filepath.Join(baseAssetDir, relativeDirHint)

		if !strings.HasPrefix(filepath.Clean(targetDir), baseAssetDir) {
			return "", fmt.Errorf("invalid relative directory hint '%s'", relativeDirHint)
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create sub-directory '%s': %w", targetDir, err)
		}
	}

	if filenameHint == "" {
		// Implement UUID generation here if needed for certain types
		return "", fmt.Errorf("filename hint cannot be empty for LocalStorage.Save")
	}
	finalFilename := filenameHint

	fullSavePath := filepath.Join(targetDir, finalFilename)

	outFile, err := os.Create(fullSavePath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file '%s': %w", fullSavePath, err)
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, data)
	if err != nil {
		outFile.Close()
		os.Remove(fullSavePath)
		return "", fmt.Errorf("failed to write data to '%s': %w", fullSavePath, err)
	}

	relativePath, err := filepath.Rel(ls.basePath, fullSavePath)
	if err != nil {
		log.Printf("media.store: Error calculating relative path for '%s' from '%s': %v", fullSavePath, ls.basePath, err)
		return "", fmt.Errorf("internal error calculating relative path: %w", err)
	}

	log.Printf("media.store: Saved asset to %s", fullSavePath)
	return filepath.ToSlash(relativePath), nil
}

func (ls *LocalStorage) Get(relativePath string) (io.ReadCloser, os.FileInfo, error) {
	fullPath, err := ls.GetFullPath(relativePath)
	if err != nil {
		return nil, nil, err
	}

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, fmt.Errorf("asset not found at '%s': %w", relativePath, err)
		}
		return nil, nil, fmt.Errorf("failed to open asset '%s': %w", relativePath, err)
	}

	info, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, fmt.Errorf("failed to stat asset '%s': %w", relativePath, err)
	}

	return file, info, nil
}

// Delete removes an asset file
func (ls *LocalStorage) Delete(relativePath string) error {
	fullPath, err := ls.GetFullPath(relativePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil // If GetFullPath determines it doesn't exist, treat as success
		}
		return err
	}

	err = os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) { // Ignore "not exist" errors
		return fmt.Errorf("failed to delete asset '%s': %w", relativePath, err)
	}
	if err == nil {
		log.Printf("media.store: Deleted asset %s", fullPath)
	}
	return nil
}

// GetFullPath calculates the absolute path and performs security check
func (ls *LocalStorage) GetFullPath(relativePath string) (string, error) {
	// clean the relative path first to prevent simple traversal tricks
	cleanRelativePath := filepath.Clean(relativePath)

	fullPath := filepath.Join(ls.basePath, cleanRelativePath)

	absFullPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for '%s': %w", relativePath, err)
	}

	if !strings.HasPrefix(absFullPath, ls.basePath) {
		return "", fmt.Errorf("invalid path: access denied for '%s'", relativePath)
	}

	return absFullPath, nil
}
