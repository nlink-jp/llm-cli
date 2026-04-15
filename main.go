package main

import "github.com/nlink-jp/llm-cli/cmd"

var version = "dev"

func main() {
	cmd.Execute(version)
}
