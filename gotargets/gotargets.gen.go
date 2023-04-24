package gotargets

type GoDistEntry struct {
	GOOS   string `json:"GOOS"`
	GOARCH string `json:"GOARCH"`
}

var KnownGoDists = []*GoDistEntry{
	{
		GOOS:   "aix",
		GOARCH: "ppc64",
	},
	{
		GOOS:   "android",
		GOARCH: "386",
	},
	{
		GOOS:   "android",
		GOARCH: "amd64",
	},
	{
		GOOS:   "android",
		GOARCH: "arm",
	},
	{
		GOOS:   "android",
		GOARCH: "arm64",
	},
	{
		GOOS:   "darwin",
		GOARCH: "amd64",
	},
	{
		GOOS:   "darwin",
		GOARCH: "arm64",
	},
	{
		GOOS:   "dragonfly",
		GOARCH: "amd64",
	},
	{
		GOOS:   "freebsd",
		GOARCH: "386",
	},
	{
		GOOS:   "freebsd",
		GOARCH: "amd64",
	},
	{
		GOOS:   "freebsd",
		GOARCH: "arm",
	},
	{
		GOOS:   "freebsd",
		GOARCH: "arm64",
	},
	{
		GOOS:   "freebsd",
		GOARCH: "riscv64",
	},
	{
		GOOS:   "illumos",
		GOARCH: "amd64",
	},
	{
		GOOS:   "ios",
		GOARCH: "amd64",
	},
	{
		GOOS:   "ios",
		GOARCH: "arm64",
	},
	{
		GOOS:   "js",
		GOARCH: "wasm",
	},
	{
		GOOS:   "linux",
		GOARCH: "386",
	},
	{
		GOOS:   "linux",
		GOARCH: "amd64",
	},
	{
		GOOS:   "linux",
		GOARCH: "arm",
	},
	{
		GOOS:   "linux",
		GOARCH: "arm64",
	},
	{
		GOOS:   "linux",
		GOARCH: "loong64",
	},
	{
		GOOS:   "linux",
		GOARCH: "mips",
	},
	{
		GOOS:   "linux",
		GOARCH: "mips64",
	},
	{
		GOOS:   "linux",
		GOARCH: "mips64le",
	},
	{
		GOOS:   "linux",
		GOARCH: "mipsle",
	},
	{
		GOOS:   "linux",
		GOARCH: "ppc64",
	},
	{
		GOOS:   "linux",
		GOARCH: "ppc64le",
	},
	{
		GOOS:   "linux",
		GOARCH: "riscv64",
	},
	{
		GOOS:   "linux",
		GOARCH: "s390x",
	},
	{
		GOOS:   "netbsd",
		GOARCH: "386",
	},
	{
		GOOS:   "netbsd",
		GOARCH: "amd64",
	},
	{
		GOOS:   "netbsd",
		GOARCH: "arm",
	},
	{
		GOOS:   "netbsd",
		GOARCH: "arm64",
	},
	{
		GOOS:   "openbsd",
		GOARCH: "386",
	},
	{
		GOOS:   "openbsd",
		GOARCH: "amd64",
	},
	{
		GOOS:   "openbsd",
		GOARCH: "arm",
	},
	{
		GOOS:   "openbsd",
		GOARCH: "arm64",
	},
	{
		GOOS:   "openbsd",
		GOARCH: "mips64",
	},
	{
		GOOS:   "plan9",
		GOARCH: "386",
	},
	{
		GOOS:   "plan9",
		GOARCH: "amd64",
	},
	{
		GOOS:   "plan9",
		GOARCH: "arm",
	},
	{
		GOOS:   "solaris",
		GOARCH: "amd64",
	},
	{
		GOOS:   "windows",
		GOARCH: "386",
	},
	{
		GOOS:   "windows",
		GOARCH: "amd64",
	},
	{
		GOOS:   "windows",
		GOARCH: "arm",
	},
	{
		GOOS:   "windows",
		GOARCH: "arm64",
	},
}
