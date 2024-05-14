-- v0 -> v1: Latest revision


-- TODO: Add email_address

CREATE TABLE portal (
    thread_id   TEXT    NOT NULL,
    receiver    TEXT    NOT NULL,
    mxid        TEXT,
    name        TEXT    NOT NULL,
    topic       TEXT    NOT NULL,
    encrypted   BOOLEAN NOT NULL DEFAULT false,
    avatar_path TEXT    NOT NULL DEFAULT '',
    avatar_hash TEXT    NOT NULL,
    avatar_url  TEXT    NOT NULL,
    name_set    BOOLEAN NOT NULL DEFAULT false,
    avatar_set  BOOLEAN NOT NULL DEFAULT false,
    topic_set   BOOLEAN NOT NULL DEFAULT false,
    revision    INTEGER NOT NULL DEFAULT 0,

    expiration_time BIGINT NOT NULL,
    relay_user_id   TEXT   NOT NULL,

    PRIMARY KEY (thread_id, receiver),
    CONSTRAINT portal_mxid_unique UNIQUE(mxid)
);

CREATE TABLE puppet (
    email_address TEXT    PRIMARY KEY,
    number        TEXT    UNIQUE,
    name          TEXT    NOT NULL,
    name_quality  INTEGER NOT NULL,
    avatar_path   TEXT    NOT NULL,
    avatar_hash   TEXT    NOT NULL,
    avatar_url    TEXT    NOT NULL,
    name_set      BOOLEAN NOT NULL DEFAULT false,
    avatar_set    BOOLEAN NOT NULL DEFAULT false,

    is_registered      BOOLEAN NOT NULL DEFAULT false,
    contact_info_set   BOOLEAN NOT NULL DEFAULT false,
    profile_fetched_at BIGINT,

    custom_mxid  TEXT,
    access_token TEXT NOT NULL,

    CONSTRAINT puppet_custom_mxid_unique UNIQUE(custom_mxid)
);

CREATE TABLE "user" (
    mxid  TEXT PRIMARY KEY,
    email_address  TEXT,
    password       TEXT,

    management_room TEXT,
    space_room      TEXT,

    CONSTRAINT user_address_unique UNIQUE(email_address)
);

CREATE TABLE user_portal (
    user_mxid       TEXT,
    portal_thread_id  TEXT,
    portal_receiver TEXT,
    last_read_ts    BIGINT  NOT NULL DEFAULT 0,
    in_space        BOOLEAN NOT NULL DEFAULT false,

    PRIMARY KEY (user_mxid, portal_thread_id, portal_receiver),
    CONSTRAINT user_portal_user_fkey FOREIGN KEY (user_mxid)
        REFERENCES "user"(mxid) ON UPDATE CASCADE ON DELETE CASCADE,
    CONSTRAINT user_portal_portal_fkey FOREIGN KEY (portal_thread_id, portal_receiver)
        REFERENCES portal(thread_id, receiver) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE message (
    sender     TEXT    NOT NULL,
    timestamp  BIGINT  NOT NULL,
    part_index INTEGER NOT NULL,

    email_address  TEXT NOT NULL,
    email_receiver TEXT NOT NULL,

    mxid    TEXT NOT NULL,
    mx_room TEXT NOT NULL,

    PRIMARY KEY (sender, timestamp, part_index, email_receiver),
    CONSTRAINT message_portal_fkey FOREIGN KEY (email_address, email_receiver)
        REFERENCES portal(thread_id, receiver) ON DELETE CASCADE ON UPDATE CASCADE,
    FOREIGN KEY (sender) REFERENCES puppet(email_address) ON DELETE CASCADE,
    CONSTRAINT message_mxid_unique UNIQUE (mxid)
);
