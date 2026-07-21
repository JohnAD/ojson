package movie

import "time"

type Movie struct {
	Title      string    `json:"title"`
	ReleasedAt time.Time `json:"released_at"`
	Tags       []string  `json:"tags"`
	Meta       Meta      `json:"meta"`
}

type Meta struct {
	Score int `json:"score"`
}
