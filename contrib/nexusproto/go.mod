module github.com/nexus-rpc/sdk-go/contrib/nexusproto

go 1.25

require (
	github.com/nexus-rpc/sdk-go v0.4.0
	github.com/stretchr/testify v1.8.4
	google.golang.org/protobuf v1.36.6
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/nexus-rpc/sdk-go => ../..
