package service

type PublicProjectDTO struct {
	Name           string `json:"name"`
	Description    string `json:"description"`
	TokenUpdatedAt string `json:"token_updated_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type PublicVersionDTO struct {
	Major       uint    `json:"major"`
	Minor       uint    `json:"minor"`
	Patch       uint    `json:"patch"`
	URL         string  `json:"url"`
	Version     string  `json:"version"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type PublicVersionSubsetDTO struct {
	Version     string  `json:"version"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
}

type PublicChangelogDTO struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	SortOrder uint   `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type PublicAnnouncementDTO struct {
	Title       string  `json:"title"`
	Content     string  `json:"content"`
	IsPinned    bool    `json:"is_pinned"`
	Status      string  `json:"status"`
	PublishedAt *string `json:"published_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type PublicVersionsResult struct {
	List     []PublicVersionDTO `json:"list"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"page_size"`
}

type PublicVersionChangelogsResult struct {
	Version    PublicVersionSubsetDTO `json:"version"`
	Changelogs []PublicChangelogDTO   `json:"changelogs"`
}

type PublicAnnouncementsResult struct {
	List     []PublicAnnouncementDTO `json:"list"`
	Total    int64                   `json:"total"`
	Page     int                     `json:"page"`
	PageSize int                     `json:"page_size"`
}
