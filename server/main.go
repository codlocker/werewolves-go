package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
	"werewolves-go/data"
	"werewolves-go/server/utils"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

/*
 * Define types for different structures used in the server path.
 */
type clientMap map[string]*actor.PID
type userMap map[string]*data.Client
type State int

/*
 * Constants to define states of the werewolf game.
 */
const (
	connect State = iota
	start
	werewolfdiscuss
	werewolfvote
	townpersondiscussion
	townspersonvote
	end
	SLen = iota
)

// The gameset variable defines the start of a game channel once the server starts receiving
// connections.
var gameSet bool

// Differents constants for the program
// TODO : Move to a config file.
var number_werewolves int = 2
var curr_state State = connect
var min_players_required int = 4
var state_start_time time.Time = time.Now()
var connection_duration time.Duration = 60 * time.Second
var werewolf_discussion_duration time.Duration = 60 * time.Second
var townsperson_discussion_duration time.Duration = 120 * time.Second
var voting_duration time.Duration = 60 * time.Second

/*
 * Server structure that initates the clients, users and logger
 * parameters required by the server.
 */
type server struct {
	clients         clientMap
	users           userMap
	werewolves      userMap
	werewolvesVotes *data.Voters
	userVotes       *data.Voters
	witches         userMap
	logger          *slog.Logger
}

/*
 * Instantiate a receiver actor for the server struct.
 */
func newServer() actor.Receiver {
	gameSet = false
	return &server{
		clients:    make(clientMap),
		users:      make(userMap),
		werewolves: make(userMap),
		witches:    make(userMap),
		logger:     slog.Default(),
	}
}

/*
 * Marks a player as dead based on the user with max votes.
 */
func (s *server) markGuyAsDead(max_voted_guy string) {

	dead_user_address := utils.GetCAddrFromUsername(s.users, max_voted_guy)
	if entry, ok := s.users[dead_user_address]; ok {
		entry.Status = false
		s.users[dead_user_address] = entry
	}

	if entry, ok := s.werewolves[dead_user_address]; ok {
		entry.Status = false
		s.werewolves[dead_user_address] = entry
	}
}

/*
 * Receive messages from other actors.
 * Initiate go channel and work through different message types.
 */
func (s *server) Receive(ctx *actor.Context) {
	if !gameSet {
		gameSet = true
		go s.gameChannel(ctx)
	}

	switch msg := ctx.Message().(type) {
	case actor.Stopped:
		s.logger.Info("Moderator has chosen to die.")
		for _, pid := range s.clients {
			ctx.Send(pid, utils.FormatMessageResponseFromServer(
				"Moderator has chosen to die. You are safe to leave."))
		}
	case *types.Message:
		if len(msg.Msg) > 0 {
			s.logger.Info("message received", "msg", msg.Msg, "from", ctx.Sender())
			s.handleMessage(ctx)
		} else {
			s.logger.Info(fmt.Sprintf("%v message was empty. hence dropped.", ctx.Sender()))
		}
	case *types.Disconnect:
		cAddr := ctx.Sender().GetAddress()
		pid, ok := s.clients[cAddr]
		if !ok {
			s.logger.Warn("unknown client disconnected", "client", cAddr)
			return
		}
		username, ok := s.users[cAddr]
		if !ok {
			s.logger.Warn("unknown user disconnected", "client", cAddr)
			return
		}
		s.logger.Info("client disconnected", "username", username, "pid", pid)
		delete(s.clients, cAddr)
		delete(s.users, cAddr)
	case *types.Connect:
		if curr_state != connect {
			ctx.Send(ctx.Sender(), &types.Message{
				Username: "server/primary",
				Msg:      "Game has already started.",
			})

			break
		}

		cAddr := ctx.Sender().GetAddress()
		if _, ok := s.clients[cAddr]; ok {
			s.logger.Warn("client already connected", "client", ctx.Sender().GetID())
			return
		}

		if _, ok := s.users[cAddr]; ok {
			s.logger.Warn("user already connected", "client", ctx.Sender().GetID())
			return
		}

		s.clients[cAddr] = ctx.Sender()
		s.users[cAddr] = data.NewClient(msg.Username, "")
		s.logger.Info("new client connected",
			"id", ctx.Sender().GetID(), "addr", ctx.Sender().GetAddress(), "sender", ctx.Sender(),
			"username", msg.Username,
		)

		s.broadcastMessage(ctx, fmt.Sprintf("%v connected", msg.Username))
	}
}

/*
 * Loops through all the states for werewolves and determines message
 * parsing across multiple states and clients.
 */
func (s *server) gameChannel(ctx *actor.Context) {
	var count int32 //counter for no. of rounds
	for {
		switch curr_state {
		case connect:
			end_time := state_start_time.Add(connection_duration)
			fmt.Printf("End time for state %v = %v\n", State.String(curr_state), end_time)
			time.Sleep(10 * time.Second)

			if len(s.users) >= min_players_required {
				s.broadcastMessage(ctx, "Minimum players reached. Ready to begin in 60 seconds!!")
				end_time := state_start_time.Add(connection_duration)
				for {
					if time.Now().After(end_time) {
						break
					}
				}

				utils.SetUpRoles(s.users, s.werewolves, number_werewolves)
				utils.PrintUsers(s.users)
				utils.SendIdentities(s.users, s.clients, ctx)
				curr_state = (curr_state + 1) % State(SLen)
			} else {
				if time.Now().After(end_time) {
					state_start_time = time.Now()
					s.broadcastMessage(ctx, "Minimum player not reached. Extending time....")
				} else {
					s.broadcastMessage(ctx, "Waiting for players....")
				}
			}
		case start:
			// Message everyone
			s.broadcastMessage(ctx, "Night falls and the town sleeps.  Everyone close your eyes")
			curr_state = (curr_state + 1) % State(SLen)
		case werewolfdiscuss:
			// Message werewolves
			count += 1 //increment counter
			s.broadcastMessage(ctx, fmt.Sprintf("========== Round: %d ==========", count))

			s.broadcastMessage(ctx, "Werewolves, open your eyes.")

			// Go to end or werewolf if number of werewolves is less than 1.
			if utils.CountWerewolvesAlive(s.users) == 1 {
				s.logger.Info(fmt.Sprintf("Not enough werewolves alive for %v state", State.String(curr_state)))
				curr_state = werewolfvote
				continue
			} else if utils.CountWerewolvesAlive(s.users) == 0 {
				curr_state = end
				continue
			}

			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to discuss", werewolf_discussion_duration))
			state_end_time := time.Now().Add(werewolf_discussion_duration)
			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			curr_state = (curr_state + 1) % State(SLen)
		case werewolfvote:
			pidList := utils.GetAliveWerewolves(s.users, s.clients)

			for _, pid := range pidList {
				msgResponse := utils.FormatMessageResponseFromServer(
					"Choose the player to kill: " + strings.Join(utils.GetListofUsernames(s.users), ","))
				ctx.Send(pid, msgResponse)
			}

			s.werewolvesVotes = data.NewVoters(
				utils.GetListofUsernames(s.users))

			s.broadcastMessage(ctx, "Werewolves, now its time to vote")
			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to vote", voting_duration))

			state_end_time := time.Now().Add(voting_duration)

			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			curr_state = (curr_state + 1) % State(SLen)
		case townpersondiscussion:
			max_voted_guy := s.werewolvesVotes.GetMaxVotedUser()
			s.broadcastMessage(ctx, "Townpeople, its time to wake up and listen to the news")
			if max_voted_guy == "" {
				s.broadcastMessage(ctx, "Townspeople, the werewolf did not feed tonight")
			} else {
				s.markGuyAsDead(max_voted_guy)
				s.broadcastMessage(ctx, fmt.Sprintf("The werewolf chose to kill %v", max_voted_guy))
			}

			s.werewolvesVotes.PrintVotes()
			s.werewolvesVotes.ClearVotes()

			// Skip to end stage if the number of users equal to 1 or when only werewolves remain.
			if (utils.CountUsersAlive(s.users) <= 1) || (utils.CountUsersAlive(s.users) == utils.CountWerewolvesAlive(s.users)) {
				curr_state = end
				continue
			}

			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to discuss", townsperson_discussion_duration))
			state_end_time := time.Now().Add(townsperson_discussion_duration)
			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			curr_state = (curr_state + 1) % State(SLen)
		case townspersonvote:
			// Initialize user voter instance.
			s.userVotes = data.NewVoters(utils.GetListofUsernames(s.users))

			s.broadcastMessage(ctx, "Townpeople, now its time for you to vote")
			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to vote", voting_duration))

			pidList := utils.GetAliveTownperson(s.users, s.clients)

			for _, pid := range pidList {
				msgResponse := utils.FormatMessageResponseFromServer(
					"Choose the player to kick out: " + strings.Join(utils.GetListofUsernames(s.users), ","))
				ctx.Send(pid, msgResponse)
			}

			state_end_time := time.Now().Add(voting_duration)
			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			max_voted_guy := s.userVotes.GetMaxVotedUser()

			if max_voted_guy == "" {
				s.broadcastMessage(ctx, "The town could not reach a consensus. No one was kicked")
			} else {
				s.markGuyAsDead(max_voted_guy)
				s.broadcastMessage(ctx, fmt.Sprintf("The town has chosen to kill %v", max_voted_guy))
			}

			s.userVotes.PrintVotes()
			curr_state = (curr_state + 1) % State(SLen)
		case end:
			// Game win scenario. If no werewolf or townperson choose to move the last state else
			if !utils.AreTownspersonAlive(s.users) && utils.AreWerewolvesAlive(s.users) {
				s.broadcastMessage(ctx, "**GAME OVER**")
				s.broadcastMessage(ctx, "Werewolves win")
				s.logger.Info("Press Ctrl + C to exit")
				return
			} else if !utils.AreWerewolvesAlive(s.users) && utils.AreTownspersonAlive(s.users) {
				s.broadcastMessage(ctx, "**GAME OVER**")
				s.broadcastMessage(ctx, "Townspeople win")
				s.logger.Info("Press Ctrl + C to exit")
				return
			} else if !utils.AreWerewolvesAlive(s.users) && !utils.AreTownspersonAlive(s.users) {
				s.broadcastMessage(ctx, "**GAME OVER**")
				s.broadcastMessage(ctx, "Everyone died")
				s.logger.Info("Press Ctrl + C to exit")
				return
			} else {
				curr_state = werewolfdiscuss
			}
		default:
			fmt.Println("State not found")
		}
	}
}

/*
 * Broadcast message sends messages to all clients.
 */
func (s *server) broadcastMessage(ctx *actor.Context, message string) {
	msgResponse := utils.FormatMessageResponseFromServer(message)
	for _, pid := range s.clients {
		ctx.Send(pid, msgResponse)
	}
}

/*
 * Handle message takes into responses from client for Message type in gRPC
 * and performs action for sending or computation accordingly.
 */
func (s *server) handleMessage(ctx *actor.Context) {
	var allowedUsers map[string]*data.Client
	var username string = ctx.Message().(*types.Message).Username

	// Check for whether the person is dead or alive
	if !s.users[ctx.Sender().GetAddress()].Status {
		ctx.Send(
			ctx.Sender(),
			utils.FormatMessageResponseFromServer("Bruh, you cant message when you are dead!"))
		return
	}

	// Do not accept messages if the game has ended
	if curr_state == State(SLen) {
		ctx.Send(
			ctx.Sender(),
			utils.FormatMessageResponseFromServer("The game has ended. Thank you for playing!"))
		return
	}

	// Check for discussion state of werewolves or witch or townsperson
	if curr_state == werewolfdiscuss || curr_state == werewolfvote {
		allowedUsers = s.werewolves
	} else {
		allowedUsers = s.users
	}

	// Only allow messages to be processed if they are in the allowed list
	if utils.IsUsernameAllowed(username, allowedUsers) {
		// Evaluate messages for voting
		if curr_state == werewolfvote || curr_state == townspersonvote {
			user_names := utils.GetListofUsernames(s.users)
			fmt.Println(user_names)
			msg := ctx.Message().(*types.Message).Msg

			if !slices.Contains(user_names, msg) {
				ctx.Send(ctx.Sender(), utils.FormatMessageResponseFromServer(
					"Please select the elements from the list only.."))
			} else {
				s.logger.Info(fmt.Sprintf("%v has chosen to kill %v",
					s.users[ctx.Sender().GetAddress()].Name,
					msg))
				if curr_state == werewolfvote {
					s.werewolvesVotes.AddVote(msg, s.users[ctx.Sender().GetAddress()].Name)
				} else if curr_state == townspersonvote {
					s.userVotes.AddVote(msg, s.users[ctx.Sender().GetAddress()].Name)
				}
			}
		}

		if curr_state != werewolfvote && curr_state != townspersonvote {
			for caddr := range allowedUsers {
				// dont send message to the place where it came from.
				pid := s.clients[caddr]

				if !pid.Equals(ctx.Sender()) {
					s.logger.Info("forwarding message", "pid", pid.ID, "addr", pid.Address, "msg", ctx.Message())
					ctx.Forward(pid)
				}
			}
		}
	} else {
		ctx.Send(ctx.Sender(), utils.FormatMessageResponseFromServer(
			fmt.Sprintf("You are not allowed to send messages in %v", State.String(curr_state))))
	}
}

// Enum to string
func (state State) String() string {
	switch state {
	case connect:
		return "connect"
	case start:
		return "start"
	case werewolfdiscuss:
		return "werewolfdiscuss"
	case werewolfvote:
		return "werewolfvote"
	case townpersondiscussion:
		return "townpersondiscussion"
	case townspersonvote:
		return "townspersonvote"
	case end:
		return "end"
	default:
		return ""
	}
}

// Entry point to the server program.
func main() {
	listenPort := flag.String("listen", "4000", "Enter the port number to open a receiver endpoint")
	flag.Parse()

	listenAddress := "127.0.0.1:" + *listenPort
	fmt.Println(listenAddress)
	rem := remote.New(listenAddress, remote.NewConfig())
	engine, err := actor.NewEngine(actor.NewEngineConfig().WithRemote(rem))

	if err != nil {
		panic(err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	serverPID := engine.Spawn(newServer, "server", actor.WithID("primary"))
	slog.Info(fmt.Sprintf("Server running at PID : %v", serverPID))

	for {
		sig := <-sigCh
		fmt.Printf("Received signal: %v\n", sig)
		if sig == os.Interrupt {
			// Create a waitgroup so we can wait until foo has been stopped gracefully
			wg := &sync.WaitGroup{}
			wg.Add(1)

			go func() {
				engine.Poison(serverPID, wg)
				time.Sleep(time.Second * 10)
				wg.Done()
			}()

			wg.Wait()

			os.Exit(1)
		}
	}
}
