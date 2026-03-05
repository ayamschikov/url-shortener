package model

import "time"

type Click struct {
	ID        int64
	URLID     int64
	IP        string
	UserAgent string
	Referer   string
	ClickedAt time.Time
}

type URLStats struct {
	Code        string
	OriginalURL string
	TotalClicks int64
}
