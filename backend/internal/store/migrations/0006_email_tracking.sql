-- Checkpoint 17: Automação de e-mail, pipeline in_process e tags de fase

CREATE TABLE IF NOT EXISTS email_accounts (
    id              TEXT PRIMARY KEY,
    provider        TEXT NOT NULL,
    address         TEXT NOT NULL,
    auth_config     BLOB NOT NULL,
    folder          TEXT NOT NULL DEFAULT 'INBOX',
    last_uid        TEXT,
    last_history_id TEXT,
    enabled         INTEGER NOT NULL DEFAULT 0,
    last_sync_at    DATETIME,
    last_error      TEXT
);

CREATE TABLE IF NOT EXISTS email_messages (
    id                     TEXT PRIMARY KEY,
    account_id             TEXT NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    message_id             TEXT NOT NULL UNIQUE,
    thread_id              TEXT,
    from_address           TEXT NOT NULL,
    from_name              TEXT,
    subject                TEXT,
    received_at            DATETIME NOT NULL,
    body_text              TEXT,
    matched_application_id INTEGER REFERENCES applications(id) ON DELETE SET NULL,
    match_confidence       REAL NOT NULL DEFAULT 0.0,
    stage_signal           TEXT,
    classified_at          DATETIME,
    link_source            TEXT
);

CREATE INDEX IF NOT EXISTS idx_email_msg_account  ON email_messages(account_id);
CREATE INDEX IF NOT EXISTS idx_email_msg_app      ON email_messages(matched_application_id);
CREATE INDEX IF NOT EXISTS idx_email_msg_thread   ON email_messages(thread_id);
CREATE INDEX IF NOT EXISTS idx_email_msg_received ON email_messages(received_at);

CREATE TABLE IF NOT EXISTS tags (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    namespace TEXT NOT NULL,
    value     TEXT NOT NULL,
    label     TEXT NOT NULL,
    UNIQUE(namespace, value)
);

CREATE INDEX IF NOT EXISTS idx_tags_namespace ON tags(namespace);

CREATE TABLE IF NOT EXISTS application_tags (
    application_id INTEGER NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    tag_id         INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (application_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_app_tags_tag ON application_tags(tag_id);

ALTER TABLE applications ADD COLUMN stage_group TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE applications ADD COLUMN stage_label TEXT;
ALTER TABLE applications ADD COLUMN stage_updated_at DATETIME;
ALTER TABLE applications ADD COLUMN stage_source TEXT NOT NULL DEFAULT 'manual';

UPDATE applications SET
    stage_group = CASE
        WHEN status = 'sent' THEN 'in_process'
        WHEN status = 'interview' THEN 'interview'
        WHEN status = 'offer' THEN 'offer'
        WHEN status = 'rejected' THEN 'rejected'
        ELSE 'pending'
    END,
    stage_updated_at = updated_at,
    stage_source = 'manual';

CREATE INDEX IF NOT EXISTS idx_app_stage_group ON applications(stage_group);
