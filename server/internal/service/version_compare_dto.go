package service

type CompareVersionItem struct {
	ID      uint64 `json:"id"`
	Version string `json:"version"`
	Status  string `json:"status"`
}

type CompareChangelogItem struct {
	ID        uint64 `json:"id"`
	VersionID uint64 `json:"version_id"`
	Version   string `json:"version"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	SortOrder uint   `json:"sort_order"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type CompareChangelogGroups struct {
	Added      []CompareChangelogItem `json:"added"`
	Changed    []CompareChangelogItem `json:"changed"`
	Fixed      []CompareChangelogItem `json:"fixed"`
	Improved   []CompareChangelogItem `json:"improved"`
	Deprecated []CompareChangelogItem `json:"deprecated"`
	Removed    []CompareChangelogItem `json:"removed"`
}

type CompareVersionsResult struct {
	FromVersion CompareVersionItem     `json:"from_version"`
	ToVersion   CompareVersionItem     `json:"to_version"`
	Versions    []CompareVersionItem   `json:"versions"`
	Changelogs  CompareChangelogGroups `json:"changelogs"`
}
