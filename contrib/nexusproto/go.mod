module github.com/nexus-rpc/sdk-go/contrib/nexusproto

go 1.23.4

require (
	github.com/nexus-rpc/sdk-go v0.1.0
	google.golang.org/protobuf v1.36.1
)

require github.com/google/uuid v1.3.0 // indirect

replace github.com/nexus-rpc/sdk-go => ../..
