CREATE DATABASE IF NOT EXISTS stan_store;
USE stan_store;

# Copy from github.com/nats-io/nats-streaming-server/scripts/mysql.db.sql

CREATE TABLE IF NOT EXISTS ServerInfo (uniquerow INT DEFAULT 1, id VARCHAR(1024), proto BLOB, version INTEGER, PRIMARY KEY (uniquerow));
CREATE TABLE IF NOT EXISTS Clients (id VARCHAR(1024), hbinbox TEXT, PRIMARY KEY (id(256)));
CREATE TABLE IF NOT EXISTS Channels (id INTEGER, name VARCHAR(1024) NOT NULL, maxseq BIGINT UNSIGNED DEFAULT 0, maxmsgs INTEGER DEFAULT 0, maxbytes BIGINT DEFAULT 0, maxage BIGINT DEFAULT 0, deleted BOOL DEFAULT FALSE, PRIMARY KEY (id), INDEX Idx_ChannelsName (name(256)));
CREATE TABLE IF NOT EXISTS Messages (id INTEGER, seq BIGINT UNSIGNED, timestamp BIGINT, size INTEGER, data BLOB, CONSTRAINT PK_MsgKey PRIMARY KEY(id, seq), INDEX Idx_MsgsTimestamp (timestamp));
CREATE TABLE IF NOT EXISTS Subscriptions (id INTEGER, subid BIGINT UNSIGNED, lastsent BIGINT UNSIGNED DEFAULT 0, proto BLOB, deleted BOOL DEFAULT FALSE, CONSTRAINT PK_SubKey PRIMARY KEY(id, subid));
CREATE TABLE IF NOT EXISTS SubsPending (subid BIGINT UNSIGNED, `row` BIGINT UNSIGNED, seq BIGINT UNSIGNED DEFAULT 0, lastsent BIGINT UNSIGNED DEFAULT 0, pending BLOB, acks BLOB, CONSTRAINT PK_MsgPendingKey PRIMARY KEY(subid, `row`), INDEX Idx_SubsPendingSeq(seq));
CREATE TABLE IF NOT EXISTS StoreLock (id VARCHAR(30), tick BIGINT UNSIGNED DEFAULT 0);

# Updates for 0.10.0
ALTER TABLE Clients ADD proto BLOB;
