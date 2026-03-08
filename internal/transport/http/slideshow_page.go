package server

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"galleryduck/internal/app/gallery"
	webpages "galleryduck/internal/web/pages"
)

func (s *Server) slideshowHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}

	values := r.URL.Query()
	playerData, err := s.buildSlideshowPlayerData(values)
	if err != nil {
		http.Error(w, "failed to build slideshow", http.StatusInternalServerError)
		return
	}

	data := webpages.SlideshowPageData{
		BackURL: buildGalleryBackURL(values),
		Theme:   s.gallery.Config().Theme,
		Player:  playerData,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := webpages.SlideshowPage(data).Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render slideshow", http.StatusInternalServerError)
	}
}

func (s *Server) slideshowItemsHandler(w http.ResponseWriter, r *http.Request) {
	if s.gallery == nil {
		http.Error(w, "gallery service unavailable", http.StatusServiceUnavailable)
		return
	}
	playerData, err := s.buildSlideshowPlayerData(r.URL.Query())
	if err != nil {
		http.Error(w, "failed to build slideshow", http.StatusInternalServerError)
		return
	}
	response := map[string]any{
		"items":      playerData.Items,
		"count":      playerData.Count,
		"scope_text": playerData.ScopeText,
		"speed_ms":   playerData.SpeedMS,
		"transition": playerData.Transition,
		"autoplay":   playerData.Autoplay,
		"loop":       playerData.Loop,
		"random":     playerData.Random,
		"fullscreen": playerData.Fullscreen,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func (s *Server) buildSlideshowPlayerData(values url.Values) (webpages.SlideshowPlayerData, error) {
	typeFilter := strings.ToLower(strings.TrimSpace(values.Get("type")))
	availableSubTypes := gallery.SupportedSubTypes(typeFilter)
	selectedSubTypes, _ := resolveSubTypeSelection(
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
	}
	if q.Sort == "" {
		q.Sort = "date"
	}
	if q.Order == "" {
		q.Order = "desc"
	}

	config := s.gallery.Config()
	speedMS := gallery.ParseQueryInt(values.Get("speed_ms"), config.Slideshow.SpeedMS)
	if speedMS < 500 {
		speedMS = 500
	}
	transition := strings.ToLower(strings.TrimSpace(values.Get("transition")))
	if transition == "" {
		transition = strings.ToLower(config.Slideshow.Transition)
	}
	if transition != "slide" {
		transition = "fade"
	}

	autoplay := parseQueryBool(values.Get("autoplay"), config.Slideshow.Autoplay)
	loop := parseQueryBool(values.Get("loop"), config.Slideshow.Loop)
	random := parseQueryBool(values.Get("random"), q.Sort == "random")
	fullscreen := parseQueryBool(values.Get("auto_fullscreen"), config.Slideshow.Fullscreen)

	items := s.gallery.QueryMediaAll(q)
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

	itemsJSONBytes, err := json.Marshal(viewItems)
	if err != nil {
		return webpages.SlideshowPlayerData{}, err
	}

	return webpages.SlideshowPlayerData{
		Items:    viewItems,
		ItemsB64: base64.StdEncoding.EncodeToString(itemsJSONBytes),
		Count:    len(viewItems),
		Query: webpages.GalleryQuery{
			Type:     q.Type,
			SubTypes: selectedSubTypes,
			Year:     strings.TrimSpace(values.Get("year")),
			Date:     q.Date,
			Search:   q.Search,
			Sort:     q.Sort,
			Order:    q.Order,
		},
		ScopeText:  buildSlideshowScopeText(q),
		SpeedMS:    speedMS,
		Transition: transition,
		Autoplay:   autoplay,
		Loop:       loop,
		Random:     random,
		Fullscreen: fullscreen,
	}, nil
}

func parseQueryBool(raw string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func buildGalleryBackQuery(values url.Values) url.Values {
	allowed := map[string]struct{}{
		"type":             {},
		"sub_type":         {},
		"subtypes_present": {},
		"year":             {},
		"date":             {},
		"search":           {},
		"sort":             {},
		"order":            {},
	}
	query := make(url.Values)
	for key, v := range values {
		if _, ok := allowed[key]; !ok {
			continue
		}
		copied := make([]string, len(v))
		copy(copied, v)
		query[key] = copied
	}
	return query
}

func buildGalleryBackURL(values url.Values) string {
	query := buildGalleryBackQuery(values).Encode()
	if query == "" {
		return "/"
	}
	return "/?" + query
}

func buildSlideshowScopeText(q gallery.Query) string {
	parts := make([]string, 0, 4)
	if q.Type == "" {
		parts = append(parts, "All media")
	} else {
		parts = append(parts, strings.ToUpper(q.Type[:1])+q.Type[1:])
	}
	if q.Year > 0 {
		parts = append(parts, fmt.Sprintf("Year=%d", q.Year))
	}
	if q.Date != "" {
		parts = append(parts, "Date="+q.Date)
	}
	if q.Sort != "" {
		parts = append(parts, "Sort="+q.Sort+"_"+q.Order)
	}
	return strings.Join(parts, ", ")
}
