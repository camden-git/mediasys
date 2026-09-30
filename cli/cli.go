// Package cli implements the management commands run as `mediasys <command>`,
// e.g. `docker compose exec backend /app/mediasys user create --admin`. Running
// the binary with no arguments starts the server instead (see main.go).
package cli

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"github.com/camden-git/mediasysbackend/config"
	"github.com/camden-git/mediasysbackend/database"
	"github.com/camden-git/mediasysbackend/handlers"
	"github.com/camden-git/mediasysbackend/models"
	"github.com/camden-git/mediasysbackend/repository"
	"golang.org/x/term"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const usage = `Usage: mediasys [command]

With no command, starts the server.

Commands:
  user create    create a user (prompts for anything not given as a flag)

Run 'mediasys <command> -h' for a command's flags.
`

// Run executes the command in args (os.Args without the program name) and
// returns the process exit code.
func Run(ctx context.Context, args []string) int {
	if len(args) >= 2 && args[0] == "user" && args[1] == "create" {
		return runUserCreate(ctx, args[2:])
	}
	switch args[0] {
	case "help", "-h", "-help", "--help":
		fmt.Print(usage)
		return 0
	}
	fmt.Fprintf(os.Stderr, "unknown command: %s\n\n%s", strings.Join(args, " "), usage)
	return 2
}

// NewUser describes a user to create with CreateUser.
type NewUser struct {
	Username  string
	FirstName string
	LastName  string
	Password  string
	// Admin assigns the Super Administrator role.
	Admin bool
}

// CreateUser validates in and creates the user, assigning the Super
// Administrator role when in.Admin is set.
func CreateUser(db *gorm.DB, in NewUser) (*models.User, error) {
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" {
		return nil, errors.New("username is required")
	}
	if msg := handlers.PasswordPolicyError(in.Password); msg != "" {
		return nil, errors.New(msg)
	}

	var roleIDs []uint
	if in.Admin {
		roleRepo := repository.NewGormRoleRepository(db)
		role, err := roleRepo.GetByName(models.SuperAdminRoleName)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// the server creates the role at startup; cover a CLI run before the first start
			if err := handlers.SyncSuperAdminRole(roleRepo); err != nil {
				return nil, err
			}
			role, err = roleRepo.GetByName(models.SuperAdminRoleName)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to load the '%s' role: %w", models.SuperAdminRoleName, err)
		}
		roleIDs = []uint{role.ID}
	}

	user := &models.User{
		Username:  in.Username,
		FirstName: strings.TrimSpace(in.FirstName),
		LastName:  strings.TrimSpace(in.LastName),
	}
	if err := user.SetPassword(in.Password); err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}
	if err := repository.NewGormUserRepository(db).CreateWithRoles(user, roleIDs); err != nil {
		if repository.IsUniqueViolation(err) {
			return nil, fmt.Errorf("username %q is already taken", in.Username)
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}
	return user, nil
}

func runUserCreate(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("user create", flag.ContinueOnError)
	fs.Usage = func() {
		fmt.Fprint(fs.Output(), `Usage: mediasys user create [flags]

Creates a user. Missing values are prompted for; the password is always read
from the terminal (or from stdin with --password-stdin), never from a flag.

Examples:
  docker compose exec backend /app/mediasys user create --admin
  echo "$PASSWORD" | docker compose exec -T backend /app/mediasys user create --username alice --password-stdin

Flags:
`)
		fs.PrintDefaults()
	}
	username := fs.String("username", "", "username to log in with")
	firstName := fs.String("first-name", "", "first name")
	lastName := fs.String("last-name", "", "last name")
	admin := fs.Bool("admin", false, "make the user a Super Administrator")
	passwordStdin := fs.Bool("password-stdin", false, "read the password from the first line of stdin")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintf(os.Stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	stdin := bufio.NewReader(os.Stdin)
	interactive := term.IsTerminal(int(os.Stdin.Fd()))

	var err error
	if *username == "" && interactive {
		fmt.Print("Username: ")
		*username, err = readLine(stdin)
		if err != nil {
			return fail(err)
		}
	}
	if strings.TrimSpace(*username) == "" {
		return fail(errors.New("username is required (--username)"))
	}

	var password string
	switch {
	case *passwordStdin:
		password, err = readLine(stdin)
	case interactive:
		password, err = promptNewPassword()
	default:
		err = errors.New("stdin is not a terminal; pass --password-stdin to read the password from it")
	}
	if err != nil {
		return fail(err)
	}
	// fail fast, before waiting on the database
	if msg := handlers.PasswordPolicyError(password); msg != "" {
		return fail(errors.New(msg))
	}

	db, err := connect(ctx)
	if err != nil {
		return fail(err)
	}
	if sqlDB, err := db.DB(); err == nil {
		defer sqlDB.Close()
	}

	user, err := CreateUser(db, NewUser{
		Username:  *username,
		FirstName: *firstName,
		LastName:  *lastName,
		Password:  password,
		Admin:     *admin,
	})
	if err != nil {
		return fail(err)
	}
	kind := "user"
	if *admin {
		kind = "Super Administrator"
	}
	fmt.Printf("Created %s %q (id %d).\n", kind, user.Username, user.ID)
	return 0
}

// connect opens the database and applies any pending migrations, so the
// command also works before the server has started for the first time.
func connect(ctx context.Context) (*gorm.DB, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	db, err := database.InitGormDB(ctx, cfg.DatabaseURL, logger.Warn)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if err := database.RunMigrations(ctx, sqlDB); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to migrate database schema: %w", err)
	}
	return db, nil
}

func promptNewPassword() (string, error) {
	fd := int(os.Stdin.Fd())

	// term.ReadPassword turns echo off and would leave it off if Ctrl-C killed
	// the process mid-prompt, so restore the terminal before exiting instead
	state, err := term.GetState(fd)
	if err != nil {
		return "", err
	}
	done := make(chan struct{})
	defer close(done)
	interrupted := make(chan os.Signal, 1)
	signal.Notify(interrupted, os.Interrupt)
	defer signal.Stop(interrupted)
	go func() {
		select {
		case <-interrupted:
			_ = term.Restore(fd, state)
			fmt.Println()
			os.Exit(130)
		case <-done:
		}
	}()

	fmt.Print("Password: ")
	first, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	fmt.Print("Confirm password: ")
	second, err := term.ReadPassword(fd)
	fmt.Println()
	if err != nil {
		return "", err
	}
	if string(first) != string(second) {
		return "", errors.New("passwords do not match")
	}
	return string(first), nil
}

// readLine returns the next line from r without its line ending.
func readLine(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		if errors.Is(err, io.EOF) {
			return "", errors.New("unexpected end of input")
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func fail(err error) int {
	fmt.Fprintln(os.Stderr, "error:", err)
	return 1
}
