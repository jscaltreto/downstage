package desktop

// Version is the human-readable build identifier shown in the About
// dialog. Set at build time via ldflags:
//
//	-X github.com/jscaltreto/downstage/internal/desktop.Version=<version>
//
// See Makefile's desktop-* targets. Falls back to "dev" for any build
// that doesn't inject a version (local `go build`, IDE runs, etc.).
var Version = "dev"
