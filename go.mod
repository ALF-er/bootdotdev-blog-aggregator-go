module github.com/alf-er/bootdotdev-blog-aggregator-go

go 1.25.5

require internal/config v1.0.0
require internal/database v1.0.0
require internal/rss v1.0.0

require (
	github.com/google/uuid v1.6.0 // indirect
	github.com/lib/pq v1.10.9 // indirect
)

replace internal/config => ./internal/config
replace internal/database => ./internal/database
replace internal/rss => ./internal/rss
