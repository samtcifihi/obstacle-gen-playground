addr := "localhost:8080"

# Start the server
serve:
    go run . -addr {{addr}}

# Start the server and open it in the default browser
open:
    go run . -addr {{addr}} -open

# Run the tests
test:
    go test ./...
