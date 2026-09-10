// Command nullrun is a cyberpunk hacking roguelike for the terminal.
package main

import (
	"context"
	"fmt"
	"os"

	"codeberg.org/anaseto/gruid"
	gtcell "codeberg.org/anaseto/gruid-tcell"
)

func main() {
	driver := gtcell.NewDriver(gtcell.Config{StyleManager: styleManager{}})
	app := gruid.NewApp(gruid.AppConfig{
		Model:  newGame(),
		Driver: driver,
	})
	if err := app.Start(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "nullrun:", err)
		os.Exit(1)
	}
}
