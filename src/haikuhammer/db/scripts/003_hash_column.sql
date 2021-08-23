CREATE TABLE IF NOT EXISTS haiku_hash (
    message_id integer,
    md5_sum    blob,
    PRIMARY KEY (message_id)
)


