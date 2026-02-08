create table epochs(
    id          serial      primary key,
    start_time  timestamp   not null,
    end_time    timestamp   not null  -- update when new epoch started
);

create table party  {
    epoch_id int foreign key references epochs(id),
    address string,
    certificate string
    domen string,
    primary key(epoch_id, address)
}

CREATE TABLE IF NOT EXISTS latest_block
(
    id INT NOT NULL UNIQUE,
    latest_block_id INT NOT NULL
);