module github.com/yyl1212/math_master/backend

replace github.com/yyl1212/math_master/schemas => ../schemas

go 1.27.1

require (
	github.com/jackc/pgx/v5 v5.11.0
	github.com/santhosh-tekuri/jsonschema/v6 v6.0.3
	github.com/yuin/goldmark v1.8.6
	github.com/yyl1212/math_master/schemas v0.0.0
)

require (
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/stretchr/testify v1.12.1 // indirect
	golang.org/x/sync v0.22.0 // indirect
	golang.org/x/text v0.41.0 // indirect
)
