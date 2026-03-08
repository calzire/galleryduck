package pages

type GalleryItem struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	SubType string `json:"sub_type"`
	Date    string `json:"date"`
	Src     string `json:"src"`
}

type GalleryQuery struct {
	Type     string
	SubTypes []string
	Year     string
	Date     string
	Search   string
	Sort     string
	Order    string
	Page     int
}

type GallerySubTypeOption struct {
	ID      string
	Value   string
	Label   string
	Checked bool
}

type GalleryPagination struct {
	Total        int
	Page         int
	HasPrev      bool
	HasNext      bool
	PrevURL      string
	NextURL      string
	NextChunkURL string
	Mode         string
	ResultType   string
}

type GalleryPageData struct {
	MediaCount        int
	LibraryCount      int
	Theme             string
	Items             []GalleryItem
	SubTypeOptions    []GallerySubTypeOption
	SubTypeAllChecked bool
	Query             GalleryQuery
	Pagination        GalleryPagination
}

type SlideshowPageData struct {
	BackURL string
	Theme   string
	Player  SlideshowPlayerData
}

type SlideshowPlayerData struct {
	Items      []GalleryItem
	ItemsB64   string
	Count      int
	Query      GalleryQuery
	ScopeText  string
	SpeedMS    int
	Transition string
	Autoplay   bool
	Loop       bool
	Random     bool
	Fullscreen bool
}

type SettingsPageData struct {
	ConfigPath   string
	LibraryPaths string
	Theme        string
	DefaultSort  string
	DefaultView  string
	Pagination   string
	SpeedMS      int
	Transition   string
	Autoplay     bool
	Loop         bool
	Fullscreen   bool
	MediaCount   int
	LocalURL     string
	LANURL       string
	LANQRURL     string
	Saved        bool
	HasError     bool
	ErrorMessage string
	LibraryCount int
}
