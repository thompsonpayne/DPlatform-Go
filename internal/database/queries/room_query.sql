-- name: GetRoom :one
SELECT * FROM rooms
WHERE id = ? LIMIT 1;

-- name: GetAllRooms :many
SELECT * FROM rooms
ORDER BY created_at;

-- name: CreateRoom :one
INSERT INTO rooms (
    id, name
) VALUES (?, ?)
RETURNING * ;

-- name: UpdateRoom :exec
update rooms
set name = ?
where id = ? ;

-- name: DeleteRoom :exec
delete from rooms where id = ? ;
