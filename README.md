# Run server
`cd server && go mod tidy && go run .`

# Run test 
`go run main.go tcp localhost:9000 <client> <msg/client>`
`go run main.go udp localhost:9001 <client> <msg/client>`
`go run main.go rudp localhost:9002 <client> <msg/client>`