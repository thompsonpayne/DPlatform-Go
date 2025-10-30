-- +goose Up
create table if not exists users (
    id text primary key,
    name text not null,
    email text not null unique,
    password text not null,  -- More descriptive name
    created_at datetime default current_timestamp
);

-- +goose Down
drop table users;
