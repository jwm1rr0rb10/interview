// Программа buildinfo печатает версию сборки.
//
//	go build -ldflags "-X main.version=v1.4.2" -o bin/app ./buildinfo
package main

import (
	"fmt"
	"runtime"
	"runtime/debug"
)

// version подставляется линкером через -ldflags "-X main.version=...".
var version = "dev"

func main() {
	fmt.Printf("version: %s\ngo: %s %s/%s\n", version, runtime.Version(), runtime.GOOS, runtime.GOARCH)

	// Команда go сама встраивает в бинарник модуль, зависимости и данные VCS.
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	fmt.Println("module:", info.Main.Path, info.Main.Version)
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision", "vcs.time", "vcs.modified", "CGO_ENABLED", "GOARCH", "-trimpath":
			fmt.Printf("%s: %s\n", s.Key, s.Value)
		}
	}
}
