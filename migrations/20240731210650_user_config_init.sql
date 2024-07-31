-- +goose Up
-- +goose StatementBegin
CREATE TABLE user_configs (
	id string PRIMARY KEY,
	timezone string
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE user_configs;
-- +goose StatementEnd
