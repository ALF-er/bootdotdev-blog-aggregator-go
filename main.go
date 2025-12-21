package main

import (
	"fmt"
	"os"
	"internal/config"
)

type state struct {
	config *config.Config
}

type command struct {
	name string
	arguments []string
}

type commands struct {
	commands map[string]func(*state, command) error
}

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	mainState := state { config: cfg }

	comms := make(map[string]func(*state, command) error)
	commandsMap := commands { commands: comms }
	commandsMap.register("login", handlerLogin)

	if len(os.Args) < 2 {
		fmt.Println("specify some command")

		os.Exit(1)
	}

	cmd := command { name: os.Args[1], arguments: os.Args[2:] }

	err = commandsMap.run(&mainState, cmd)
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	os.Exit(0)
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for login command - user name")
	}

	err := s.config.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Printf("successfull login")

	return nil
}

func (c *commands) run(s *state, cmd command) error {
	handler, exst := c.commands[cmd.name]
	if !exst {
		return fmt.Errorf("no such command")
	}

	err := handler(s, cmd)
	if err != nil {
		return err
	}

	return nil
}

func (c *commands) register(name string, f func(*state, command) error) {
	c.commands[name] = f
}
