package global

// Version is overridden at build time:
//
//	-ldflags "-X github.com/Streamlet/notesite/global.Version=$(git describe --tags --exact-match 2>/dev/null || git rev-parse --short HEAD)"

var (
	ProductName = "notesite"
	Version     = "latest"
)
