-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE pull_requests (
    id SERIAL PRIMARY KEY,
    github_id VARCHAR(255) UNIQUE NOT NULL,
    repo_full_name VARCHAR(255) NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    url TEXT NOT NULL DEFAULT '',
    number INTEGER NOT NULL DEFAULT 0,
    author VARCHAR(255) NOT NULL DEFAULT '',
    base_branch VARCHAR(255) NOT NULL DEFAULT '',
    head_branch VARCHAR(255) NOT NULL DEFAULT '',
    head_sha VARCHAR(255) NOT NULL DEFAULT '',
    worktree_path TEXT NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE reviews (
    id SERIAL PRIMARY KEY,
    pull_request_id INTEGER NOT NULL REFERENCES pull_requests(id) ON DELETE CASCADE,
    commit_sha VARCHAR(255) NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    overall_verdict VARCHAR(50) NOT NULL DEFAULT '',
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    executor_name VARCHAR(100) NOT NULL DEFAULT '',
    process_logs TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT idx_pr_commit UNIQUE (pull_request_id, commit_sha)
);

CREATE INDEX idx_reviews_pull_request_id ON reviews(pull_request_id);

CREATE TABLE review_comments (
    id SERIAL PRIMARY KEY,
    review_id INTEGER NOT NULL REFERENCES reviews(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL DEFAULT '',
    line_start INTEGER NOT NULL DEFAULT 0,
    line_end INTEGER NOT NULL DEFAULT 0,
    body TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_review_comments_review_id ON review_comments(review_id);

CREATE TABLE repo_configs (
    id SERIAL PRIMARY KEY,
    repo_full_name VARCHAR(255) UNIQUE NOT NULL,
    local_path TEXT NOT NULL DEFAULT '',
    cli VARCHAR(100) NOT NULL DEFAULT '',
    extra_rules TEXT,
    active BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE cli_configs (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT true
);

INSERT INTO cli_configs (name, description, active) VALUES ('claude-code', 'Claude Code CLI agent', true) ON CONFLICT DO NOTHING;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS cli_configs;
DROP TABLE IF EXISTS repo_configs;
DROP TABLE IF EXISTS review_comments;
DROP TABLE IF EXISTS reviews;
DROP TABLE IF EXISTS pull_requests;
