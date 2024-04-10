package data

type Client struct {
	Name string
	Role string
}

func NewClient(name string, role string) *Client {
	return &Client{Name: name, Role: role}
}
