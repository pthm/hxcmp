package main

import (
	"log"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	hxcmpecho "github.com/pthm/hxcmp/adapters/echo"
	"github.com/pthm/hxcmp/examples/echo-counter/components"
)

func main() {
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// Mount hxcmp components
	reg := hxcmpecho.Mount(e)
	components.Init(reg)

	// Page route
	e.GET("/", func(c echo.Context) error {
		return hxcmpecho.Render(c, page())
	})

	log.Fatal(e.Start(":8080"))
}
