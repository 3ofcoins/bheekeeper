package main

import "github.com/3ofcoins/bheekeeper/packer"
import "github.com/mitchellh/packer/packer/plugin"

func main() {
	server, err := plugin.Server()
	if err != nil {
		panic(err)
	}
	server.RegisterBuilder(new(packer.Builder))
	server.Serve()
}
