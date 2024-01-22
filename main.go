package main

import (
	"os"

	"github.com/Harshjha3006/golang-blockchain/cli"
)

func main() {
	defer os.Exit(0)
	cli := cli.Cmd{}

	cli.Run()

}
