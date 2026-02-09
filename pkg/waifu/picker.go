package waifu

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strings"
)

// supportedExtensions is the set of image file extensions recognized by the
// picker. Extensions are stored lowercase without a leading dot.
var supportedExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".webp": true,
	".bmp":  true,
}

// PickRandom selects a random image file from the given directory. It filters
// out non-image files, hidden files (dot-prefixed), and subdirectories.
// Returns the absolute path to the selected image.
func PickRandom(dir string) (string, error) {
	images, err := ListImages(dir)
	if err != nil {
		return "", err
	}
	if len(images) == 0 {
		return "", fmt.Errorf("no image files found in %s", dir)
	}

	idx := rand.IntN(len(images))
	return images[idx], nil
}

// ListImages returns absolute paths for all valid image files in the given
// directory. It filters out hidden files, directories, and files with
// unsupported extensions. The returned paths are sorted for deterministic
// output (random selection is done by the caller).
func ListImages(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read image directory: %w", err)
	}

	var images []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip hidden files.
		if strings.HasPrefix(name, ".") {
			continue
		}
		ext := strings.ToLower(filepath.Ext(name))
		if supportedExtensions[ext] {
			images = append(images, filepath.Join(dir, name))
		}
	}

	return images, nil
}
