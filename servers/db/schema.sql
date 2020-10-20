create table if not exists USERS (
    id int not null auto_increment primary key,
    Email varchar(254) not null,
    PassHash varchar(72) not null,
    UserName varchar(255) not null,
    FirstName varchar(64) not null,
    LastName varchar(128) not null,
    PhotoURL varchar(255) not null
);

CREATE INDEX userIndex
ON USERS (Email, UserName)