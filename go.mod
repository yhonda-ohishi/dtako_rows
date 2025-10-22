module github.com/yhonda-ohishi/dtako_rows/v3

go 1.25.1

require (
	github.com/joho/godotenv v1.5.1
	github.com/yhonda-ohishi/db_service v0.0.0-20250101000000-000000000000
	google.golang.org/grpc v1.76.0
	google.golang.org/protobuf v1.36.10
)

require (
	golang.org/x/net v0.46.0 // indirect
	golang.org/x/sys v0.37.0 // indirect
	golang.org/x/text v0.30.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20251014184007-4626949a642f // indirect
)

replace github.com/yhonda-ohishi/db_service => ../db_service
