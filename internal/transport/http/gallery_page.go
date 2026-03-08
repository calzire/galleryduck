package server

import (
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"galleryduck/internal/app/gallery"
	webpages "galleryduck/internal/web/pages"
)

func (s *Server) homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if s.gallery != nil && s.gallery.NeedsSetup() {
		http.Redirect(w, r, "/settings", http.StatusFound)
		return
	}

	data := s.buildGalleryPageData(r.URL.Query())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.GalleryPage(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render gallery", http.StatusInternalServerError)
	}
}

func (s *Server) mediaListFragmentHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	data := s.buildGalleryPageData(r.URL.Query())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.GalleryResults(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render media list", http.StatusInternalServerError)
	}
}

func (s *Server) mediaChunkFragmentHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	data := s.buildGalleryPageData(r.URL.Query())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.GalleryEndlessChunk(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render media chunk", http.StatusInternalServerError)
	}
}

func (s *Server) mediaSubTypesFragmentHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	data := s.buildGalleryPageData(r.URL.Query())
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.GallerySubTypeOptions(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render subtype options", http.StatusInternalServerError)
	}
}

func (s *Server) buildGalleryPageData(values url.Values) webpages.GalleryPageData {
	if s.gallery == nil {
		return webpages.GalleryPageData{
			MediaCount:        0,
			LibraryCount:      0,
			Theme:             "system",
			Items:             []webpages.GalleryItem{},
			SubTypeOptions:    []webpages.GallerySubTypeOption{},
			SubTypeAllChecked: true,
			Query: webpages.GalleryQuery{
				Sort:  "date",
				Order: "desc",
				Page:  1,
			},
			Pagination: webpages.GalleryPagination{Page: 1},
		}
	}

	typeFilter := strings.ToLower(strings.TrimSpace(values.Get("type")))
	availableSubTypes := gallery.SupportedSubTypes(typeFilter)
	selectedSubTypes, allSelected := resolveSubTypeSelection(
		availableSubTypes,
		values["sub_type"],
		strings.TrimSpace(values.Get("subtypes_present")) == "1",
	)

	q := gallery.Query{
		Type:     typeFilter,
		SubTypes: selectedSubTypes,
		Year:     gallery.ParseQueryInt(values.Get("year"), 0),
		Date:     strings.TrimSpace(values.Get("date")),
		Search:   strings.TrimSpace(values.Get("search")),
		Sort:     strings.TrimSpace(values.Get("sort")),
		Order:    strings.TrimSpace(values.Get("order")),
		Page:     gallery.ParseQueryInt(values.Get("page"), 1),
		PageSize: 24,
	}
	if q.Sort == "" {
		q.Sort = "date"
	}
	if q.Order == "" {
		q.Order = "desc"
	}
	if q.Page < 1 {
		q.Page = 1
	}

	items, total := s.gallery.QueryMedia(q)
	viewItems := make([]webpages.GalleryItem, 0, len(items))
	for _, item := range items {
		viewItems = append(viewItems, webpages.GalleryItem{
			Name:    item.Name,
			Type:    item.Type,
			SubType: item.SubType,
			Date:    item.Date.Format("2006-01-02"),
			Src:     "/api/media/file?path=" + url.QueryEscape(item.Path),
		})
	}

	lastPage := 1
	if total > 0 {
		lastPage = (total + q.PageSize - 1) / q.PageSize
	}
	hasPrev := q.Page > 1
	hasNext := q.Page < lastPage

	cfg := s.gallery.Config()
	paginationMode := strings.ToLower(strings.TrimSpace(cfg.PaginationMode))
	if paginationMode != "endless" {
		paginationMode = "page"
	}

	data := webpages.GalleryPageData{
		MediaCount:        total,
		LibraryCount:      len(s.gallery.LibraryPaths()),
		Theme:             cfg.Theme,
		Items:             viewItems,
		SubTypeOptions:    buildSubTypeOptions(availableSubTypes, selectedSubTypes),
		SubTypeAllChecked: allSelected,
		Query: webpages.GalleryQuery{
			Type:     q.Type,
			SubTypes: selectedSubTypes,
			Year:     strings.TrimSpace(values.Get("year")),
			Date:     q.Date,
			Search:   q.Search,
			Sort:     q.Sort,
			Order:    q.Order,
			Page:     q.Page,
		},
		Pagination: webpages.GalleryPagination{
			Total:   total,
			Page:    q.Page,
			HasPrev: hasPrev,
			HasNext: hasNext,
			Mode:    paginationMode,
		},
	}

	if hasPrev {
		prevQ := cloneQuery(values)
		prevQ.Set("page", strconv.Itoa(q.Page-1))
		data.Pagination.PrevURL = "/media/list?" + prevQ.Encode()
	}
	if hasNext {
		nextQ := cloneQuery(values)
		nextQ.Set("page", strconv.Itoa(q.Page+1))
		data.Pagination.NextURL = "/media/list?" + nextQ.Encode()
		data.Pagination.NextChunkURL = "/media/chunk?" + nextQ.Encode()
	}

	return data
}

func resolveSubTypeSelection(available []string, queryValues []string, hasSubTypesInput bool) ([]string, bool) {
	if len(available) == 0 {
		return []string{}, true
	}

	availableSet := make(map[string]struct{}, len(available))
	for _, subType := range available {
		availableSet[subType] = struct{}{}
	}

	selectedSet := make(map[string]struct{}, len(queryValues))
	for _, raw := range queryValues {
		subType := strings.ToLower(strings.TrimSpace(raw))
		if subType == "" {
			continue
		}
		if _, ok := availableSet[subType]; ok {
			selectedSet[subType] = struct{}{}
		}
	}

	// Default: if no explicit/valid selection was provided, select all.
	if len(selectedSet) == 0 {
		// Explicit subtype UI submit with no values means the user deselected all.
		if hasSubTypesInput && len(queryValues) == 0 {
			return []string{}, false
		}

		selected := make([]string, len(available))
		copy(selected, available)
		return selected, true
	}

	selected := make([]string, 0, len(selectedSet))
	for subType := range selectedSet {
		selected = append(selected, subType)
	}
	sort.Strings(selected)
	return selected, len(selected) == len(available)
}

func buildSubTypeOptions(available []string, selected []string) []webpages.GallerySubTypeOption {
	selectedSet := make(map[string]struct{}, len(selected))
	for _, subType := range selected {
		selectedSet[subType] = struct{}{}
	}

	options := make([]webpages.GallerySubTypeOption, 0, len(available))
	for _, subType := range available {
		_, checked := selectedSet[subType]
		options = append(options, webpages.GallerySubTypeOption{
			ID:      "subtype-" + subType,
			Value:   subType,
			Label:   strings.ToUpper(subType),
			Checked: checked,
		})
	}
	return options
}

func cloneQuery(v url.Values) url.Values {
	out := make(url.Values, len(v))
	for k, vals := range v {
		copied := make([]string, len(vals))
		copy(copied, vals)
		out[k] = copied
	}
	return out
}
