package main

import (
	"fmt"
	"log"
	"net"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/squaremo/ambergreen/common/backends"
	"github.com/squaremo/ambergreen/common/data"
)

const DEFAULT_GROUP = data.InstanceGroup("default")

type addOpts struct {
	backend *backends.Backend

	spec
}

func (opts *addOpts) addCommandTo(top *cobra.Command) {
	addCmd := &cobra.Command{
		Use:   "service <name> <IP address> <port> [options]",
		Short: "define a service",
		Long:  "Define a service, optionally giving a default specification for instances belonging to that service.",
		Run:   opts.run,
	}
	opts.addSpecVars(addCmd)
	top.AddCommand(addCmd)
}

func (opts *addOpts) run(cmd *cobra.Command, args []string) {
	if len(args) < 3 {
		exitWithErrorf("Expected arguments <name>, <IP address>, <port>")
	}
	serviceName := args[0]
	port, err := strconv.Atoi(args[2])
	if err != nil {
		exitWithErrorf("Invalid port number: " + err.Error())
	}
	ip := net.ParseIP(args[1])
	if ip == nil {
		exitWithErrorf("invalid IP address: ", args[1])
	}

	instSpecs := make(map[data.InstanceGroup]data.InstanceSpec)
	if spec, err := opts.makeSpec(); err == nil {
		if spec != nil {
			instSpecs[DEFAULT_GROUP] = *spec
		}
	} else {
		exitWithErrorf("Unable to extract spec from opitions: ", err)
	}

	err = opts.backend.AddService(serviceName, data.Service{
		Address:       args[1],
		Port:          port,
		Protocol:      opts.protocol,
		InstanceSpecs: instSpecs,
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Added service:", serviceName)
}
