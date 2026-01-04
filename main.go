package main

import (
	"context"
	"database/sql"
	"fmt"
	"internal/config"
	"internal/database"
	"internal/rss"
	"os"
	"strconv"
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
	commandsMap.register("agg", middlewareLoggedIn(handlerAggregate))
	commandsMap.register("addfeed", middlewareLoggedIn(handlerAddFeed))
	commandsMap.register("feeds", handlerFeeds)
	commandsMap.register("follow", middlewareLoggedIn(handlerFollow))
	commandsMap.register("following", middlewareLoggedIn(handlerFollowing))
	commandsMap.register("unfollow", middlewareLoggedIn(handlerUnfollow))
	commandsMap.register("browse", middlewareLoggedIn(handlerBrowse))

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

func middlewareLoggedIn(handler func(s *state, cmd command, user database.User) error) func(*state, command) error {
	return func(s *state, cmd command) error {
		user, err := s.db.GetUserByName(context.Background(), s.cfg.CurrentUserName)
		if err != nil {
			return err
		}

		handler(s, cmd, user)

		return nil
	}
}

func scrapeFeeds(s *state) error {
	feedToFetch, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		fmt.Println("some error while getting next feed to fetch")

		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), feedToFetch.ID)
	if err != nil {
		fmt.Println("some error while marking feed as fetched")

		return err
	}

	feed, err := rss.FetchFeed(context.Background(), feedToFetch.Url)
	if err != nil {
		fmt.Println("some error while fetching feed")

		return err
	}

	fmt.Printf("\nScrapping feed \"%s\"\n", feed.Channel.Title)

	for i, item := range feed.Channel.Item {
		fmt.Printf("\t%d. %s\n", i, item.Title)

		publishedAt, err := time.Parse(time.RFC1123Z, item.PubDate)
		if err != nil {
			fmt.Println("some error while parsing publishing time of the post")
			fmt.Println(err)

			return err
		}

		s.db.CreatePost(context.Background(), database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Title: sql.NullString{ String: item.Title, Valid: true },
			Url: item.Link,
			Description: sql.NullString{ String: item.Description, Valid: true },
			PublishedAt: publishedAt,
			FeedID: feedToFetch.ID,
		})
		// if err != nil {
		// 	fmt.Println("some error while saving post")
		// 	fmt.Println(err)
		//
		// 	return err
		// }
	}

	return nil
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

func handlerAggregate(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for agg command - time between requests")
	}

	timeBetweenRequests, err := time.ParseDuration(cmd.arguments[0])
	if err != nil {
		fmt.Println("error while parsing argument as time duration")

		return err
	}

	fmt.Printf("Collecting feeds every %s\n\n", timeBetweenRequests.String())

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		err = scrapeFeeds(s)

		if err != nil {
			return err
		}
	}
}

func handlerAddFeed(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 2 {
		return fmt.Errorf("there should be two arguments for addfeed command - feed name and feed url")
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

	_, err = s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("some error while following the feed")

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

func handlerFollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for follow command - url of feed to follow")
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.arguments[0])
	if err != nil {
		fmt.Println("some error while retrieving feed to follow")

		return err
	}

	feed_follow, err := s.db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("some error while following the feed")

		return err
	}

	fmt.Printf("successfully following feed: %s, by: %s\n", feed_follow.FeedName, feed_follow.UserName)

	return nil
}

func handlerFollowing(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 0 {
		return fmt.Errorf("there shouldn't be any arguments for following command")
	}

	user_feed_follows, err := s.db.GetFeedFollowsForUser(context.Background(), currentUser.ID)
	if err != nil {
		fmt.Println("some error while retrieving feeds followed by current user")

		return err
	}

	for _, follows := range user_feed_follows {
		fmt.Printf("%s\n", follows.FeedName)
	}

	return nil
}

func handlerUnfollow(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for unfollow command - url of feed to unfollow")
	}

	feed, err := s.db.GetFeedByURL(context.Background(), cmd.arguments[0])
	if err != nil {
		fmt.Println("some error while retrieving feed to unfollow")

		return err
	}

	err = s.db.RemoveFeedFollowsForUser(context.Background(), database.RemoveFeedFollowsForUserParams{
		UserID: currentUser.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		fmt.Println("some error while removing feed follow")

		return err
	}

	fmt.Println("successfull unfollow")

	return nil
}

func handlerBrowse(s *state, cmd command, currentUser database.User) error {
	if len(cmd.arguments) != 1 {
		return fmt.Errorf("there should be one argument for browse command - limit of posts")
	}

	limit, err := strconv.ParseInt(cmd.arguments[0], 10, 32)
	if err != nil {
		limit = 2
	}
	posts, err := s.db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: currentUser.ID,
		Limit: int32(limit),
	})
	if err != nil {
		fmt.Println("some error while retrieving users posts")

		return err
	}

	for _, post := range posts {
		fmt.Printf("Post \"%s\":\n", post.Title.String)
		fmt.Printf("\t%s\n", post.Description.String)
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
