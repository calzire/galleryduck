package gallery

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var supportedTypes = map[string]string{
	"jpg":  "image",
	"jpeg": "image",
	"png":  "image",
	"gif":  "image",
	"webp": "image",
	"bmp":  "image",
	"heic": "image",

	"mp4":  "video",
	"mov":  "video",
	"mkv":  "video",
	"avi":  "video",
	"webm": "video",

	"mp3":  "audio",
	"wav":  "audio",
	"flac": "audio",
	"m4a":  "audio",
	"aac":  "audio",
	"ogg":  "audio",
}

func SupportedSubTypes(typeFilter string) []string {
	typeFilter = strings.ToLower(strings.TrimSpace(typeFilter))
	subTypes := make([]string, 0, len(supportedTypes))
	for subType, mediaType := range supportedTypes {
		if typeFilter != "" && mediaType != typeFilter {
			continue
		}
		subTypes = append(subTypes, subType)
	}
	sort.Strings(subTypes)
	return subTypes
}

var (
	filenameDateDash = regexp.MustCompile(`(?i)(\d{4})-(\d{2})-(\d{2})`)
	filenameDateComp = regexp.MustCompile(`(?i)(\d{4})(\d{2})(\d{2})`)
)

func scanMedia(roots []string) ([]MediaItem, error) {
	items := make([]MediaItem, 0, 256)
	seen := map[string]struct{}{}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}

		absRoot, err := filepath.Abs(root)
		if err != nil {
			continue
		}

		info, err := os.Stat(absRoot)
		if err != nil {
			continue
		}

		if !info.IsDir() {
			item, ok := mediaItemFromFile(absRoot, info)
			if !ok {
				continue
			}
			if _, exists := seen[item.Path]; exists {
				continue
			}
			seen[item.Path] = struct{}{}
			items = append(items, item)
			continue
		}

		err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if strings.HasPrefix(name, ".") && path != absRoot {
					return filepath.SkipDir
				}
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return nil
			}
			item, ok := mediaItemFromFile(path, info)
			if !ok {
				return nil
			}
			if _, exists := seen[item.Path]; exists {
				return nil
			}
			seen[item.Path] = struct{}{}
			items = append(items, item)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return items, nil
}

func mediaItemFromFile(path string, info fs.FileInfo) (MediaItem, bool) {
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(path)), ".")
	mediaType, ok := supportedTypes[ext]
	if !ok {
		return MediaItem{}, false
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return MediaItem{}, false
	}

	derivedDate := deriveDateFromName(filepath.Base(path), info.ModTime())

	return MediaItem{
		Path:    absPath,
		Name:    filepath.Base(path),
		Type:    mediaType,
		SubType: ext,
		Date:    derivedDate,
		ModTime: info.ModTime(),
	}, true
}

func deriveDateFromName(name string, fallback time.Time) time.Time {
	if date, ok := parseDateFromMatches(filenameDateDash.FindStringSubmatch(name)); ok {
		return date
	}
	if date, ok := parseDateFromMatches(filenameDateComp.FindStringSubmatch(name)); ok {
		return date
	}
	return fallback
}

func parseDateFromMatches(matches []string) (time.Time, bool) {
	if len(matches) < 4 {
		return time.Time{}, false
	}
	value := matches[1] + "-" + matches[2] + "-" + matches[3]
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, false
	}
	return date, true
}
