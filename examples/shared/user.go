package shared

import "strconv"

// User a shared user struct between server & client
type User struct {
	ID        int
	Firstname string
	Lastname  string
	Residence string
}

// NewTestUser returns a new user based on the id, it's for testing
func NewTestUser(id int) *User {
	return &User{
		ID:        id,
		Firstname: "Firstname " + strconv.Itoa(id),
		Lastname:  "Lastname " + strconv.Itoa(id),
		Residence: "Town " + strconv.Itoa(id),
	}
}
