CREATE TABLE IF NOT EXISTS channel_config (
    channel_id INTEGER,
    flags      INTEGER, -- 1 ReactToHaiku; 2 ReactToNonHaiku; 4 DeleteNonHaiku; 8 ExplainNonHaiku; 16 ServeRandomHaiku
    PRIMARY KEY (channel_id)
);

CREATE TABLE IF NOT EXISTS guild_config (
    guild_id          INTEGER,
    flags             INTEGER,  -- 1 ReactToHaiku; 2 ReactToNonHaiku; 4 DeleteNonHaiku; 8 ExplainNonHaiku; 16 ServeRandomHaiku
    positive_reacts   TEXT,
    negative_reacts   TEXT,
    PRIMARY KEY (guild_id)
);