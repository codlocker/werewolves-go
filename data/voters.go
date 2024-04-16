package data

import (
	"math"
)

type Voters struct {
	user_vote   map[string]int
	voted_users map[string]bool
}

func NewVoters(users []string) *Voters {
	v := &Voters{
		user_vote:   make(map[string]int),
		voted_users: make(map[string]bool),
	}

	for _, user := range users {
		v.user_vote[user] = 0
	}

	return v
}

func (voters *Voters) GetMaxVotedUser() string {
	var dead_users []string
	max_votes := math.MinInt

	for _, vote := range voters.user_vote {
		if vote > max_votes {
			max_votes = vote
		}
	}

	for user, vote := range voters.user_vote {
		if vote == max_votes {
			dead_users = append(dead_users, user)
		}
	}
	if len(dead_users) > 1 {
		return ""
	} else {
		return dead_users[0]
	}
}

func (voters *Voters) AddVote(user string) bool {
	_, ok := voters.user_vote[user]
	if ok && !voters.voted_users[user] {
		voters.user_vote[user]++
		voters.voted_users[user] = true
		return true
	} else {
		return false
	}
}

func (voter *Voters) ClearVotes() {
	for user := range voter.user_vote {
		voter.user_vote[user] = 0
	}

	voter.voted_users = nil
}
