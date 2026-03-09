package gallery

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Config struct {
	LibraryPaths         []string        `json:"library_paths"`
	Port                 int             `json:"port"`
	Theme                string          `json:"theme"`
	DefaultSort          string          `json:"default_sort"`
	DefaultView          string          `json:"default_view"`
	PaginationMode       string          `json:"pagination_mode"`
	Slideshow            SlideshowConfig `json:"slideshow"`
	SettingsPasswordSalt string          `json:"settings_password_salt,omitempty"`
	SettingsPasswordHash string          `json:"settings_password_hash,omitempty"`
}

type SlideshowConfig struct {
	SpeedMS    int    `json:"speed_ms"`
	Transition string `json:"transition"`
	Autoplay   bool   `json:"autoplay"`
	Loop       bool   `json:"loop"`
	Fullscreen bool   `json:"fullscreen"`
}

type MediaItem struct {
	Path    string    `json:"path"`
	Name    string    `json:"name"`
	Type    string    `json:"type"`
	SubType string    `json:"sub_type"`
	Date    time.Time `json:"date"`
	ModTime time.Time `json:"mod_time"`
}

type Query struct {
	Type     string
	SubTypes []string
	Year     int
	Date     string
	Search   string
	Sort     string
	Order    string
	Page     int
	PageSize int
}

type Service struct {
	mu         sync.RWMutex
	config     Config
	configPath string
	execDir    string
	defaultDir string
	media      []MediaItem
}

func NewService() (*Service, error) {
	execPath, err := os.Executable()
	if err != nil {
		return nil, err
	}
	execDir := filepath.Dir(execPath)
	defaultDir := resolveDefaultLibraryDir(execDir)

	configPath, err := resolveConfigPath()
	if err != nil {
		return nil, err
	}

	s := &Service{
		configPath: configPath,
		execDir:    execDir,
		defaultDir: defaultDir,
	}

	if err := s.loadOrCreateConfig(); err != nil {
		return nil, err
	}
	if err := s.rescanLocked(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *Service) ConfigPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.configPath
}

func (s *Service) Config() Config {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

func (s *Service) MediaCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.media)
}

func (s *Service) Rescan() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rescanLocked()
}

func (s *Service) LibraryPaths() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]string, len(s.config.LibraryPaths))
	copy(out, s.config.LibraryPaths)
	return out
}

func (s *Service) QueryMedia(q Query) ([]MediaItem, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	filtered := s.filterAndSortMediaLocked(q)
	total := len(filtered)

	page := q.Page
	if page < 1 {
		page = 1
	}
	pageSize := q.PageSize
	if pageSize <= 0 {
		pageSize = 24
	}
	if pageSize > 200 {
		pageSize = 200
	}

	start := (page - 1) * pageSize
	if start >= total {
		return []MediaItem{}, total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total
}

func (s *Service) QueryMediaAll(q Query) []MediaItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.filterAndSortMediaLocked(q)
}

func (s *Service) filterAndSortMediaLocked(q Query) []MediaItem {
	filtered := make([]MediaItem, 0, len(s.media))
	search := strings.ToLower(strings.TrimSpace(q.Search))
	typeFilter := strings.ToLower(strings.TrimSpace(q.Type))
	subTypeFilter := make(map[string]struct{}, len(q.SubTypes))
	for _, subType := range q.SubTypes {
		subType = strings.ToLower(strings.TrimSpace(subType))
		if subType == "" {
			continue
		}
		subTypeFilter[subType] = struct{}{}
	}

	var exactDate time.Time
	exactDateSet := false
	if strings.TrimSpace(q.Date) != "" {
		if d, err := time.Parse("2006-01-02", q.Date); err == nil {
			exactDate = d
			exactDateSet = true
		}
	}

	for _, item := range s.media {
		if typeFilter != "" && item.Type != typeFilter {
			continue
		}
		if len(subTypeFilter) > 0 {
			if _, ok := subTypeFilter[item.SubType]; !ok {
				continue
			}
		}
		if q.Year > 0 && item.Date.Year() != q.Year {
			continue
		}
		if exactDateSet && !sameDate(item.Date, exactDate) {
			continue
		}
		if search != "" {
			name := strings.ToLower(item.Name)
			path := strings.ToLower(item.Path)
			if !strings.Contains(name, search) && !strings.Contains(path, search) {
				continue
			}
		}
		filtered = append(filtered, item)
	}

	sortMedia(filtered, q.Sort, q.Order)
	return filtered
}

func (s *Service) AvailableSubTypes(typeFilter string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	typeFilter = strings.ToLower(strings.TrimSpace(typeFilter))
	subTypes := make([]string, 0, 16)
	seen := make(map[string]struct{}, 16)
	for _, item := range s.media {
		if typeFilter != "" && item.Type != typeFilter {
			continue
		}
		if _, ok := seen[item.SubType]; ok {
			continue
		}
		seen[item.SubType] = struct{}{}
		subTypes = append(subTypes, item.SubType)
	}
	sort.Strings(subTypes)
	return subTypes
}

func sameDate(a, b time.Time) bool {
	ay, am, ad := a.Date()
	by, bm, bd := b.Date()
	return ay == by && am == bm && ad == bd
}

func sortMedia(items []MediaItem, sortBy, order string) {
	sortBy = strings.ToLower(strings.TrimSpace(sortBy))
	order = strings.ToLower(strings.TrimSpace(order))
	if order == "" {
		order = "desc"
	}

	if sortBy == "random" {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		rng.Shuffle(len(items), func(i, j int) {
			items[i], items[j] = items[j], items[i]
		})
		return
	}

	less := func(i, j int) bool {
		a, b := items[i], items[j]
		switch sortBy {
		case "name":
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case "type":
			if a.Type == b.Type {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
			return a.Type < b.Type
		case "date", "":
			if a.Date.Equal(b.Date) {
				return strings.ToLower(a.Name) < strings.ToLower(b.Name)
			}
			return a.Date.Before(b.Date)
		default:
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if order == "asc" {
			return less(i, j)
		}
		return less(j, i)
	})
}

func (s *Service) NeedsSetup() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.media) == 0
}

func (s *Service) UpdateConfig(cfg Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = s.normalizeConfig(cfg)
	if err := s.saveConfigLocked(); err != nil {
		return err
	}
	return s.rescanLocked()
}

func (s *Service) loadOrCreateConfig() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(filepath.Dir(s.configPath), 0o755); err != nil {
		return err
	}

	content, err := os.ReadFile(s.configPath)
	if errors.Is(err, os.ErrNotExist) {
		s.config = s.defaultConfig()
		return s.saveConfigLocked()
	}
	if err != nil {
		return err
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return err
	}
	// Backward-compatible default for older configs that don't have slideshow.fullscreen.
	if !bytes.Contains(content, []byte(`"fullscreen"`)) {
		cfg.Slideshow.Fullscreen = true
	}
	s.config = s.normalizeConfig(cfg)
	return s.saveConfigLocked()
}

func (s *Service) saveConfigLocked() error {
	encoded, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return err
	}
	encoded = append(encoded, '\n')
	return os.WriteFile(s.configPath, encoded, 0o644)
}

func (s *Service) defaultConfig() Config {
	return Config{
		LibraryPaths:   []string{s.defaultDir},
		Port:           8787,
		Theme:          "system",
		DefaultSort:    "date_desc",
		DefaultView:    "grid",
		PaginationMode: "endless",
		Slideshow: SlideshowConfig{
			SpeedMS:    3000,
			Transition: "fade",
			Autoplay:   true,
			Loop:       true,
			Fullscreen: true,
		},
	}
}

func (s *Service) normalizeConfig(cfg Config) Config {
	if len(cfg.LibraryPaths) == 0 {
		cfg.LibraryPaths = []string{s.defaultDir}
	}

	uniquePaths := make([]string, 0, len(cfg.LibraryPaths))
	seen := map[string]struct{}{}
	for _, p := range cfg.LibraryPaths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := filepath.Abs(p)
		if err != nil {
			abs = p
		}
		if _, exists := seen[abs]; exists {
			continue
		}
		seen[abs] = struct{}{}
		uniquePaths = append(uniquePaths, abs)
	}

	if len(uniquePaths) == 0 {
		uniquePaths = []string{s.defaultDir}
	}
	if len(uniquePaths) == 1 && isTemporaryBuildDir(uniquePaths[0]) {
		uniquePaths[0] = s.defaultDir
	}
	cfg.LibraryPaths = uniquePaths
	if cfg.Port < 1 || cfg.Port > 65535 {
		cfg.Port = 8787
	}

	if cfg.Theme == "" {
		cfg.Theme = "system"
	}
	if cfg.DefaultSort == "" {
		cfg.DefaultSort = "date_desc"
	}
	if cfg.DefaultView == "" {
		cfg.DefaultView = "grid"
	}
	if cfg.PaginationMode == "" {
		cfg.PaginationMode = "endless"
	}
	if cfg.Slideshow.SpeedMS <= 0 {
		cfg.Slideshow.SpeedMS = 3000
	}
	if cfg.Slideshow.Transition == "" {
		cfg.Slideshow.Transition = "fade"
	}
	cfg.SettingsPasswordSalt = strings.TrimSpace(cfg.SettingsPasswordSalt)
	cfg.SettingsPasswordHash = strings.TrimSpace(cfg.SettingsPasswordHash)
	if cfg.SettingsPasswordSalt == "" || cfg.SettingsPasswordHash == "" {
		cfg.SettingsPasswordSalt = ""
		cfg.SettingsPasswordHash = ""
	}

	return cfg
}

func resolveConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		base := os.Getenv("APPDATA")
		if base == "" {
			var err error
			base, err = os.UserConfigDir()
			if err != nil {
				return "", err
			}
		}
		return filepath.Join(base, "galleryduck", "config.json"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".galleryduck", "config.json"), nil
}

func resolveDefaultLibraryDir(execDir string) string {
	if override := strings.TrimSpace(os.Getenv("GALLERYDUCK_LIBRARY_PATH")); override != "" {
		if abs, err := filepath.Abs(override); err == nil {
			return abs
		}
		return override
	}

	if isTemporaryBuildDir(execDir) {
		if wd, err := os.Getwd(); err == nil {
			if abs, err := filepath.Abs(wd); err == nil {
				return abs
			}
			return wd
		}
	}
	return execDir
}

func isTemporaryBuildDir(dir string) bool {
	cleanDir := strings.ToLower(filepath.Clean(dir))
	tempRoot := strings.ToLower(filepath.Clean(os.TempDir()))

	if !strings.Contains(cleanDir, "go-build") {
		return false
	}
	return strings.Contains(cleanDir, tempRoot)
}

func (s *Service) rescanLocked() error {
	items, err := scanMedia(s.config.LibraryPaths)
	if err != nil {
		return err
	}
	slices.SortFunc(items, func(a, b MediaItem) int {
		switch {
		case a.Date.After(b.Date):
			return -1
		case a.Date.Before(b.Date):
			return 1
		default:
			return strings.Compare(a.Path, b.Path)
		}
	})
	s.media = items
	return nil
}

func ParseQueryInt(value string, fallback int) int {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	v, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return v
}
