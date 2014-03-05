package main

import (
	"fmt"
	"github.com/gonuts/commander"
	"github.com/typester/go-pit"
	"log"
	"net/url"
	"os"
)

var mainCmd = &commander.Command{
	UsageLine: os.Args[0],
}

var client *Client

func init() {
	mainCmd.Subcommands = []*commander.Command{
		listCmd,
		infoCmd,
		configCmd,
		makeRegisterCmd(),
		startCmd,
		stopCmd,
		forceStopCmd,
		makeRemoveCmd(),
		publicKeyCmd,
		userDataCmd,
		sshCmd,
	}
}

func main() {
	profile, err := pit.Get(
		"urume.config",
		pit.Requires{"endpoint": "API endpoint (http://....)"},
	)
	if err != nil {
		errExit(err)
	}
	endpoint, err := url.Parse((*profile)["endpoint"])
	if err != nil {
		errExit(err)
	}

	log.Println("endpoint:", endpoint.String())

	client = NewClient(endpoint)

	if err := mainCmd.Dispatch(os.Args[1:]); err != nil {
		errExit(err)
	}
}

func errExit(err error) {
	fmt.Println(err)
	os.Exit(1)
}
