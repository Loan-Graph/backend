package auth

type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	WalletAddress string
}
