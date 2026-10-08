-- +goose up
ALTER TABLE users
ADD COLUMN hashed_password TEXT NOT NULL DEFAULT '';

ALTER TABLE users
ALTER COLUMN hashed_password DROP DEFAULT;


-- +goose down
ALTER TABLE users
DROP COLUMN hashed_password;
