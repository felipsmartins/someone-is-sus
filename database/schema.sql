CREATE TABLE games
(
    id   INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL
);

-- reported players
CREATE TABLE players
(
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id   TEXT UNIQUE NOT NULL,
    game_id     INT         NOT NULL,
    reported_by TEXT,
    reported_at TEXT        NOT NULL,
    CONSTRAINT fk_name FOREIGN KEY (game_id) REFERENCES games (id)
);

INSERT INTO games (name)
VALUES ('Team Fortress 2');
