package main

import (
	"context"
	"fmt"
	"os"

	"github.com/fiatjaf/gitstr"
)

func main() {
	if err := gitstr.App.Run(context.Background(), os.Args); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
