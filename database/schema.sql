-- Table for Retrospective
CREATE TABLE IF NOT EXISTS retrospectives (
    id          TEXT PRIMARY KEY,
    name        TEXT,
    description TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Table for Question
CREATE TABLE IF NOT EXISTS questions (
    id      TEXT PRIMARY KEY,
    text    TEXT,
    retrospective_id TEXT,
    FOREIGN KEY(retrospective_id) REFERENCES retrospectives(id)
);

-- Table for Answer
CREATE TABLE IF NOT EXISTS answers (
    id       TEXT PRIMARY KEY,
    text     TEXT,
    position INTEGER,
    question_id TEXT,
    FOREIGN KEY(question_id) REFERENCES questions(id)
);
