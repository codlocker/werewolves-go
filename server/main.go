package main

import (
	"flag"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"
	"werewolves-go/data"
	"werewolves-go/server/utils"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

type clientMap map[string]*actor.PID
type userMap map[string]*data.Client

var gameSet bool

var states = [7]string{"connect", "start", "werewolfdiscuss", "werewolfvote", "townpersondiscussion", "townspersonvote", "end"}
var number_werewolves int = 1
var curr_state int = 0
var min_players_required int = 2
var state_start_time time.Time = time.Now()
var connection_duration time.Duration = 60 * time.Second
var werewolf_discussion_duration time.Duration = 60 * time.Second
var townsperson_discussion_duration time.Duration = 120 * time.Second
var voting_duration time.Duration = 60 * time.Second

type server struct {
	clients         clientMap
	users           userMap
	werewolves      userMap
	werewolvesVotes *data.Voters
	userVotes       *data.Voters
	witches         userMap
	logger          *slog.Logger
}

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

func (s *server) Receive(ctx *actor.Context) {
	if !gameSet {
		gameSet = true
		go s.gameChannel(ctx)
	}

	switch msg := ctx.Message().(type) {
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
			s.logger.Warn("unknown client disconnected", "client", pid.Address)
			return
		}
		username, ok := s.users[cAddr]
		if !ok {
			s.logger.Warn("unknown user disconnected", "client", pid.Address)
			return
		}
		s.logger.Info("client disconnected", "username", username)
		delete(s.clients, cAddr)
		delete(s.users, username.Name)
	case *types.Connect:
		if curr_state != 0 {
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

func (s *server) gameChannel(ctx *actor.Context) {
	for {
		time.Sleep(10 * time.Second)
		switch states[curr_state] {
		case "connect":
			end_time := state_start_time.Add(connection_duration)
			fmt.Printf("End time for state %v = %v\n", states[curr_state], end_time)

			if len(s.users) >= min_players_required {
				s.broadcastMessage(ctx, "Minimum players reached. Ready to begin in 60 seconds!!")
				end_time := state_start_time.Add(connection_duration)
				for {
					if time.Now().After(end_time) {
						break
					}
				}

				utils.SetUpRoles(s.users, s.werewolves, number_werewolves)
				fmt.Println(s.users)
				curr_state = (curr_state + 1) % len(states)
			} else {
				if time.Now().After(end_time) {
					state_start_time = time.Now()
					s.broadcastMessage(ctx, "Minimum player not reached. Extending time....")
				} else {
					s.broadcastMessage(ctx, "Waiting for players....")
				}
			}
		case "start":
			// Message everyone
			s.broadcastMessage(ctx, "Night falls and the town sleeps.  Everyone close your eyes")
			curr_state = (curr_state + 1) % len(states)
		case "werewolfdiscuss":
			// Message werewolves
			s.broadcastMessage(ctx, "Werewolves, open your eyes.")
			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to discuss", werewolf_discussion_duration))
			pidList := utils.GetAliveWerewolves(s.users, s.clients)

			for _, pid := range pidList {
				msgResponse := utils.FormatMessageResponseFromServer(
					"Choose the player to kill: " + strings.Join(utils.GetListofUsernames(s.users), ","))
				ctx.Send(pid, msgResponse)
			}

			state_end_time := time.Now().Add(werewolf_discussion_duration)
			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			curr_state = (curr_state + 1) % len(states)
		case "werewolfvote":
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

			curr_state = (curr_state + 1) % len(states)
		case "townpersondiscussion":
			max_voted_guy := s.werewolvesVotes.GetMaxVotedUser()
			s.broadcastMessage(ctx, "Townpeople, its time to wake up and listen to the news")
			if max_voted_guy == "" {
				s.broadcastMessage(ctx, "Townspeople, the werewolf did not feed tonight")
			} else {
				s.markGuyAsDead(max_voted_guy)
				s.broadcastMessage(ctx, fmt.Sprintf("The werewolf chose to kill %v", max_voted_guy))
			}

			s.werewolvesVotes.ClearVotes()
			s.broadcastMessage(ctx, fmt.Sprintf("You have %v time to discuss", townsperson_discussion_duration))
			state_end_time := time.Now().Add(townsperson_discussion_duration)
			for {
				if time.Now().After(state_end_time) {
					break
				}
			}

			curr_state = (curr_state + 1) % len(states)
		case "townspersonvote":
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

			curr_state = (curr_state + 1) % len(states)
		case "end":
			max_voted_guy := s.userVotes.GetMaxVotedUser()

			if max_voted_guy == "" {
				s.broadcastMessage(ctx, "The town could not reach a consensus. No one was kicked")
			} else {
				s.markGuyAsDead(max_voted_guy)
				s.broadcastMessage(ctx, fmt.Sprintf("The town has chosen to kill %v", max_voted_guy))
			}

			// Game win scenario. If no werewolf or townperson choose to move the last state else
			if !utils.AreTownspersonAlive(s.users) && utils.AreWerewolvesAlive(s.users) {
				s.broadcastMessage(ctx, "Werewolves win")
			} else if !utils.AreWerewolvesAlive(s.users) && utils.AreTownspersonAlive(s.users) {
				s.broadcastMessage(ctx, "Townsperson win")
			} else {
				curr_state = 2
			}

			s.userVotes.ClearVotes()
		default:
			fmt.Println("State not found")
		}
	}
}

func (s *server) broadcastMessage(ctx *actor.Context, message string) {
	msgResponse := utils.FormatMessageResponseFromServer(message)
	for _, pid := range s.clients {
		ctx.Send(pid, msgResponse)
	}
}

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
	if curr_state == len(states)-1 {
		ctx.Send(
			ctx.Sender(),
			utils.FormatMessageResponseFromServer("The game has ended. Thank you for playing!"))
		return
	}

	// Check for discussion state of werewolves or witch or townsperson
	if curr_state == 2 || curr_state == 3 {
		allowedUsers = s.werewolves
	} else {
		allowedUsers = s.users
	}

	// Only allow messages to be processed if they are in the allowed list
	if utils.IsUsernameAllowed(username, allowedUsers) {
		// Evaluate messages for voting
		if curr_state == 3 || curr_state == 5 {
			user_names := utils.GetListofUsernames(s.users)
			fmt.Println(user_names)
			msg := ctx.Message().(*types.Message).Msg

			if !slices.Contains(user_names, msg) {
				ctx.Send(ctx.Sender(), utils.FormatMessageResponseFromServer(
					"Please select the elements from the list only.."))
			} else {
				fmt.Printf("%v has chosen to kill %v",
					s.users[ctx.Sender().GetAddress()].Name,
					msg)
				if curr_state == 3 {
					s.werewolvesVotes.AddVote(msg)
				} else if curr_state == 5 {
					s.userVotes.AddVote(msg)
				}
			}
		}

		if curr_state != 3 && curr_state != 5 {
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
			fmt.Sprintf("You are not allowed to send messages in %v", states[curr_state])))
	}
}

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

	engine.Spawn(newServer, "server", actor.WithID("primary"))

	select {}
}
