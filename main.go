package main

import (
	"context"
	"database/sql"
	"fmt"
	"internal/config"
	"internal/database"
	"internal/rss"
	"os"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

type state struct {
	cfg *config.Config
	db *database.Queries
}

type command struct {
	name string
	arguments []string
}

type commands struct {
	commands map[string]func(*state, command) error
}

func main() {
	mainState := state {}

	cfg, err := config.Read()
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	mainState.cfg = cfg

	db, err := sql.Open("postgres", cfg.DbUrl)
	if err != nil {
		fmt.Println(err)

		os.Exit(1)
	}

	mainState.db = database.New(db)

	commandsMap := commands { commands: make(map[string]func(*state, command) error) }
	commandsMap.register("register", handlerRegister)
	commandsMap.register("login", handlerLogin)
	commandsMap.register("reset", handlerReset)
	commandsMap.register("users", handlerUsers)
	commandsMap.register("agg", handlerAggregate)
	commandsMap.register("addfeed", handlerAddFeed)
	commandsMap.register("feeds", handlerFeeds)

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

func handlerRegister(s *state, cmd command) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for register command - user name")
	}

	err := s.cfg.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	user, err := s.db.CreateUser(context.Background(), database.CreateUserParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.arguments[0],
	})
	if err != nil {
		return err
	}

	fmt.Println("successfull register")
	fmt.Println(user)

	return nil
}

func handlerLogin(s *state, cmd command) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for login command - user name")
	}

	_, err := s.db.GetUserByName(context.Background(), cmd.arguments[0])
	if err != nil {
		fmt.Println("no such user")

		return err
	}

	err = s.cfg.SetUser(cmd.arguments[0])
	if err != nil {
		return err
	}

	fmt.Println("successfull login")

	return nil
}

func handlerReset(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("there shouldn't be any arguments for reset command")
	}

	err := s.db.Reset(context.Background())
	if err != nil {
		fmt.Println("reset failed")

		return err
	}

	fmt.Println("successfull reset")

	return nil
}

func handlerUsers(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("there shouldn't be any arguments for users command")
	}

	users, err := s.db.GetUsers(context.Background())
	if err != nil {
		fmt.Println("some error while retrieving users")

		return err
	}

	for _, user := range users {
		msg := fmt.Sprintf("* %s", user)

		if user == s.cfg.CurrentUserName {
			msg = fmt.Sprintf("%s (current)", msg)
		}

		fmt.Println(msg)
	}

	return nil
}

func handlerAggregate(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("there shouldn't be any arguments for agg command")
	}

	feed, err := rss.FetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		fmt.Println("some error while fetching feed")

		return err
	}

	fmt.Println(*feed)

	return nil
}

func handlerAddFeed(s *state, cmd command) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("there should be two arguments for addfeed command - feed name and feed url")
	}

	currentUser, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	feed, err := s.db.CreateFeed(context.Background(), database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Name: cmd.arguments[0],
		Url: cmd.arguments[1],
		UserID: currentUser.ID,
	})
	if err != nil {
		return err
	}

	fmt.Println("successfull add feed")
	fmt.Println(feed)

	return nil
}

func handlerFeeds(s *state, cmd command) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("there shouldn't be any arguments for feeds command")
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		fmt.Println("some error while retrieving feeds")

		return err
	}

	for _, feed := range feeds {
		msg := fmt.Sprintf("Name: %s, URL: %s", feed.Name, feed.Url)

		user, err := s.db.GetUserById(context.Background(), feed.UserID)
		if err != nil {
			fmt.Println("some error while retrieving user who has added feed")

			return err
		}

		fmt.Printf("%s, User: %s\n", msg, user.Name)
	}

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
