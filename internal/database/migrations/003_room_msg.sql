-- +goose Up
create index idx_messages_room_id on messages (room_id);
create index idx_messages_room_id_id on messages (room_id, id);

-- +goose Down
drop index if exists idx_messages_room_id_id;
drop index if exists idx_messages_room_id;
