CREATE TABLE IF NOT EXISTS haiku (
    guild_id       INTEGER,
    channel_id     INTEGER,
    message_id     INTEGER,
    author_mention TEXT,
    content        TEXT,
    PRIMARY KEY (guild_id, channel_id, message_id)
)