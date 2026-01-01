-- name: GetPlayer :one
SELECT *
FROM players
WHERE id = ?
LIMIT 1;

-- name: ListPlayers :many
SELECT *
FROM players;

-- name: RegisterPlayer :one
INSERT INTO players (player_id, game_id, reported_by, reported_at)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: RemovePlayer :exec
DELETE
FROM players
WHERE id = ?;