-- +goose Up
create table if not exists rooms (
    id text primary key,
    name text not null,
    created_at datetime default current_timestamp
);

create table if not exists room_users (
    room_id text not null,
    user_id text not null,
    joined_at datetime default current_timestamp,
    primary key (room_id, user_id),
    foreign key (room_id) references rooms (id) on delete cascade,
    foreign key (user_id) references users (id) on delete cascade
);

create table if not exists messages (
    id text primary key,
    room_id text not null,
    user_id text not null,
    content text not null,
    created_at datetime default current_timestamp,
    foreign key (room_id) references rooms (id) on delete cascade,
    foreign key (user_id) references users (id) on delete cascade
);

-- +goose Down
drop table rooms;
drop table room_users;
drop table messages;
