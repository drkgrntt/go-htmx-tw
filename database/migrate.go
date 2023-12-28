package database

var schema = `
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS blogs (
	id UUID PRIMARY KEY NOT NULL DEFAULT uuid_generate_v4(),
	date TIMESTAMP,
	title VARCHAR(255),
	content TEXT
);
`

func Migrate() {
	db.MustExec(schema)
}
