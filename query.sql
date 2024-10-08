-- name: GetWord :one
SELECT * FROM word
WHERE word = ? LIMIT 1;

-- name: Listword :many
SELECT * FROM word;

-- name: CreateWord :one
INSERT INTO word (
  word, zh_trans, added_count, lookup_count
) VALUES (
  ?, ?, 0, 0
)
RETURNING *;

-- name: CountWord :one
SELECT COUNT(*) FROM word WHERE word = ?;


-- name: AddWordCount :exec
UPDATE word
set added_count=added_count+1
WHERE word = ?;

-- name: DeleteWord :exec
DELETE FROM word
WHERE word = ?;