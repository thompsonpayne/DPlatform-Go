-- name: GetInitalMessages :many
select
    messages.id as message_id,
    messages.content, messages.created_at,
    users.id as user_id,
    users.name as user_name,
    users.email as user_email,
    rooms.id as room_id,
    rooms.name as room_name
from messages
join users on messages.user_id = users.id
join rooms on messages.room_id = rooms.id
where room_id = ?
order by messages.created_at desc
limit 15;

-- name: GetPaginatedMessages :many
select
    messages.id as message_id,
    messages.content, messages.created_at,
    users.id as user_id, users.name as user_name, users.email as user_email,
    rooms.id as room_id,
    rooms.name as room_name
from messages
join users on messages.user_id = users.id
join rooms on messages.room_id = rooms.id
where room_id = ? and datetime (messages.created_at) < datetime (?)
order by messages.created_at desc
limit 15;

-- name: CreateMessage :one
insert into messages (id, room_id, user_id, content)
values (?, ?, ?, ?)
returning * ;

-- name: UpdateMessage :exec
update messages
set content = ?
where id = ? ;

-- name: DeleteMessage :exec
delete from messages where id = ? ;
