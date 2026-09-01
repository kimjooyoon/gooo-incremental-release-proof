package proof

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

func CollectInventory(root string) Inventory {
	var result Inventory
	result.RootReadmeExcluded = true
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if path == filepath.Join(root, ".git") {
			return filepath.SkipDir
		}
		if strings.HasPrefix(path, filepath.Join(root, ".git")+string(filepath.Separator)) {
			return nil
		}
		if info.IsDir() {
			if path != root {
				result.DescendantDirs++
			}
			return nil
		}
		if !info.Mode().IsRegular() || path == filepath.Join(root, "README.md") {
			return nil
		}
		result.RegularFiles++
		switch filepath.Ext(path) {
		case ".go":
			result.GoFiles++
			result.GoPhysicalLines += physicalLines(path)
		case ".gooo":
			result.GoooFiles++
			result.GoooPhysicalLines += physicalLines(path)
		}
		return nil
	})
	return result
}

func physicalLines(path string) int64 {
	file, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var lines int64
	for scanner.Scan() {
		lines++
	}
	return lines
}
