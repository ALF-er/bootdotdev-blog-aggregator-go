-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE posts (
	id uuid DEFAULT gen_random_uuid() PRIMARY KEY,
	created_at TIMESTAMP NOT NULL,
	updated_at TIMESTAMP NOT NULL,
	title TEXT,
	url TEXT UNIQUE NOT NULL,
	description TEXT,
	published_at TIMESTAMP NOT NULL,
	feed_id uuid NOT NULL REFERENCES feeds(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE posts;
