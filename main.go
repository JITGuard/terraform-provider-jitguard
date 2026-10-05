package main

import (
	"context"
	"flag"
	"log"

	"github.com/JITGuard/terraform-provider-jitguard/internal/provider"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
)

var (
	// version is set at build time via -ldflags "-X main.version=...".
	version string = "dev"
)

func main() {
	var debug bool
	flag.BoolVar(&debug, "debug", false, "set to true to run the provider with support for debuggers like delve")
	flag.Parse()

	opts := providerserver.ServeOpts{
		Address: "registry.terraform.io/JITGuard/jitguard",
		Debug:   debug,
	}
	if err := providerserver.Serve(context.Background(), provider.New(version), opts); err != nil {
		log.Fatal(err.Error())
	}
}
