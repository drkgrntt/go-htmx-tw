package database

var schema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS blogs (
	id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
	date TIMESTAMP,
	title VARCHAR(255),
	content TEXT,
	published_at TIMESTAMP,
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Each row represents one inbound message from an outside sender to one of
-- our @<domain> aliases (e.g. hey@derekgarnett.com). Its id doubles as the
-- token embedded in the relay+<id>@<domain> reply-to address we hand back to
-- the personal inbox, so a reply from there can be traced back to who it
-- should actually go out to and from which alias.
CREATE TABLE IF NOT EXISTS email_relays (
	id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
	alias_address VARCHAR(255) NOT NULL,
	external_address VARCHAR(255) NOT NULL,
	external_name VARCHAR(255) NOT NULL DEFAULT '',
	subject VARCHAR(998) NOT NULL DEFAULT '',
	message_id VARCHAR(998) NOT NULL DEFAULT '',
	created_at TIMESTAMP NOT NULL DEFAULT NOW(),
	last_used_at TIMESTAMP NOT NULL DEFAULT NOW()
);
`

func Migrate() {
	db.MustExec(schema)
}
