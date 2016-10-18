package shared

// User just a shared test struct
type User struct {
	Username string
	ClientID uint64 // int is uint64 or float64 on standard json? wtf...
}
