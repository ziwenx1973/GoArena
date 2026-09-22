package game

type Choice string

const (
	Rock     Choice = "rock"
	Paper    Choice = "paper"
	Scissors Choice = "scissors"
)

func ValidChoice(c Choice) bool { return c == Rock || c == Paper || c == Scissors }

// Judge returns 1 for player 1, -1 for player 2 and 0 for a draw.
// Callers must validate both choices first.
func Judge(a, b Choice) int {
	if a == b {
		return 0
	}
	if a == Rock && b == Scissors || a == Scissors && b == Paper || a == Paper && b == Rock {
		return 1
	}
	return -1
}
