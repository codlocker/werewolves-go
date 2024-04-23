package utils

import (
	"fmt"
	"log/slog"
	"math/rand/v2"
	"slices"
	"werewolves-go/data"
	"werewolves-go/types"

	"github.com/anthdm/hollywood/actor"
)

func AreWerewolvesAlive(users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Status && user.Role == "werewolf" {
			return true
		}
	}

	return false
}

func IsWitchAlive(users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Status && user.Role == "witch" {
			return true
		}
	}

	return false
}

func IsUserAlive(user *data.Client) bool {
	return user.Status
}

func CountWerewolvesAlive(users map[string]*data.Client) int {
	count := 0
	for _, user := range users {
		if user.Status && user.Role == "werewolf" {
			count++
		}
	}

	return count
}

func AreTownspersonAlive(users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Status && (user.Role == "townsperson" || user.Role == "witch") {
			return true
		}
	}

	return false
}

func CountUsersAlive(users map[string]*data.Client) int {
	count := 0
	for _, user := range users {
		if user.Status {
			count++
		}
	}

	return count
}

func GetAliveWerewolves(users map[string]*data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Role == "werewolf" && data.Status {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}

func GetAliveWitch(users map[string]*data.Client, clients map[string]*actor.PID) *actor.PID {

	var pid *actor.PID
	for cAddr, data := range users {
		if data.Role == "witch" && data.Status {
			pid = clients[cAddr]
		}
	}

	return pid
}

func SendIdentities(users map[string]*data.Client, clients map[string]*actor.PID, ctx *actor.Context) {
	for caddr, pid := range clients {
		role := users[caddr].Role
		ctx.Send(pid, FormatMessageResponseFromServer(fmt.Sprintf("========== You are a %v =========", role)))
	}
}

func GetAliveTownperson(users map[string]*data.Client, clients map[string]*actor.PID) []*actor.PID {

	var pidList []*actor.PID
	for cAddr, data := range users {
		if data.Status {
			pidList = append(pidList, clients[cAddr])
		}
	}

	return pidList
}

func GetListofUsernames(users map[string]*data.Client) []string {
	var userList []string
	for _, data := range users {
		if data.Status {
			userList = append(userList, data.Name)
		}
	}

	return userList
}

func GetCAddrFromUsername(users map[string]*data.Client, username string) string {
	for caddr, user := range users {
		if user.Name == username {
			return caddr
		}
	}

	return ""
}

func FormatMessageResponseFromServer(message string) *types.Message {
	msgResponse := &types.Message{
		Username: "server/primary",
		Msg:      message,
	}

	return msgResponse
}

func IsUsernameAllowed(username string, users map[string]*data.Client) bool {
	for _, user := range users {
		if user.Name == username {
			return true
		}
	}

	return false
}

func SetUpRoles(users map[string]*data.Client, witches map[string]*data.Client, werewolves map[string]*data.Client, number_of_werewolves int) {
	//create a list of unique random numbers. length of list is equal to the number of werewolves you want in the
	//game.
	var listRand []int
	for i := 0; i < number_of_werewolves; {
		randNum := rand.IntN(10000) % len(users)
		if slices.Contains(listRand, randNum) {
			//randNum already present in our list - so re-run the random number generation again
		} else {
			listRand = append(listRand, randNum)
			i++
		}
	}

	// assign the witch role to a player - checks whether that player is already assigned to be a werewolf.
	var witchRand int = 0
	for {
		witchRand := rand.IntN(10000) % len(users)
		if slices.Contains(listRand, witchRand) {
			//randNum already present in our list - so re-run the random number generation again
		} else {
			break
		}
	}

	user_names := GetListofUsernames(users)
	slog.Info(fmt.Sprintf("listRand = %v :These indices in %v will be the werewolves.\n", listRand, user_names))

	for i, userName := range user_names {
		var assignWerewolf = false
		var assignWitch = false
		for _, randNum := range listRand {
			if randNum == i {
				assignWerewolf = true
			}
		}
		if witchRand == i {
			assignWitch = true
		}
		caddr := GetCAddrFromUsername(users, user_names[i])
		if assignWerewolf {
			if users[caddr].Role == "" { //performing an additional sanity check with this line
				if entry, ok := users[caddr]; ok {
					entry.Role = "werewolf"
					users[caddr] = entry
					werewolves[caddr] = entry
				}
				slog.Info(userName + " has been assigned to be a werewolf")
			}
		} else if assignWitch {
			if users[caddr].Role == "" { //performing an additional sanity check with this line
				if entry, ok := users[caddr]; ok {
					entry.Role = "witch"
					users[caddr] = entry
					witches[caddr] = entry
				}
				slog.Info(userName + " has been assigned to be a witch")
			}
		} else {
			if users[caddr].Role == "" { //performing an additional sanity check with this line
				if entry, ok := users[caddr]; ok {
					entry.Role = "townsperson"
					users[caddr] = entry
				}
				slog.Info(userName + " has been assigned to be a townsperson")
			}
		}
	}
}

func PrintUsers(users map[string]*data.Client) {
	fmt.Println("Print Users")
	for user, data := range users {
		fmt.Printf(
			"%v has been assigned %v username, has role %v with alive status %v\n",
			user,
			data.Name,
			data.Role,
			data.Status)
	}
}
