CREATE TABLE messages (
    userid integer not null,
    chatid integer not null,
    timestamp bigint not null,
    username varchar(15) not null,
    text varchar(100) not null
);