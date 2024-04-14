package data

type Client struct {
	Name   string
	Role   string
	Status bool
}

func NewClient(name string, role string) *Client {
	return &Client{Name: name, Role: role, Status: true}
}
