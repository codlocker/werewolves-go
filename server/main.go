package main

import (
	"flag"
	"fmt"
	"log/slog"
	"time"
	"werewolves-go/data"
	"werewolves-go/server/utils"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
	"github.com/anthdm/hollywood/remote"
)

type clientMap map[string]*actor.PID
type userMap map[string]data.Client

var gameSet bool

var states = [5]string{"connect", "start", "discuss", "vote", "end"}
var curr_state int = 0
var min_players_required int = 2
var state_start_time time.Time = time.Now()
var connection_duration time.Duration = 60 * time.Second

type server struct {
	clients clientMap
	users   userMap
	logger  *slog.Logger
}

func newServer() actor.Receiver {
	gameSet = false

	return &server{
		clients: make(clientMap),
		users:   make(userMap),
		logger:  slog.Default(),
	}
}

func (s *server) createRole() string {
	if !utils.DoesWerewolfExist(s.users) {
		return "werewolf"
	} else if !utils.DoesWitchExist(s.users) {
		return "witch"
	} else {
		return "townsperson"
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
			slog.Info(fmt.Sprintf("%v message was empty. hence dropped.", ctx.Sender()))
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
		s.users[cAddr] = data.Client{Name: msg.Username, Role: s.createRole()}
		slog.Info("new client connected",
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
				s.broadcastMessage(ctx, "Minimum players reached. ready to begin!!")
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
			s.broadcastMessage(ctx, "Night falls and the town sleeps.  Everyone close your eyes")
			s.broadcastMessage(ctx, "Werewolves, open your eyes.")
			pidList := utils.GetWerewolves(s.users, s.clients)

			for _, pid := range pidList {
				msgResponse := &types.Message{
					Username: "server/primary",
					Msg:      "Choose the player to kill",
				}
				ctx.Send(pid, msgResponse)
			}

			for {
			}
		default:
			fmt.Println("State not found")
		}
	}
}

func (s *server) broadcastMessage(ctx *actor.Context, message string) {
	msgResponse := &types.Message{
		Username: "server/primary",
		Msg:      message,
	}
	for _, pid := range s.clients {
		ctx.Send(pid, msgResponse)
	}
}

func (s *server) handleMessage(ctx *actor.Context) {
	for _, pid := range s.clients {
		// dont send message to the place where it came from.
		if !pid.Equals(ctx.Sender()) {
			s.logger.Info("forwarding message", "pid", pid.ID, "addr", pid.Address, "msg", ctx.Message())
			ctx.Forward(pid)
		}
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
