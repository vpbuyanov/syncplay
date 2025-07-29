create table if not exists "rooms"
(
    id         uuid                                      not null primary key,
    created_at timestamp without time zone default now() not null
);

create table if not exists "chat_messages"
(
    id         serial primary key,
    room_id    uuid references rooms (id),
    sender     text,
    text       text,
    created_at timestamp without time zone default now() not null
);

