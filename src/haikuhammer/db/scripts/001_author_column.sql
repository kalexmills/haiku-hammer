ALTER TABLE haiku RENAME COLUMN author_mention TO author_id;
UPDATE haiku set author_id = substr(author_id, 3, 18);