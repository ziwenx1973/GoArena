package tests

import (
	"github.com/ziwenx1973/GoArena/internal/game"
	"testing"
)

func TestJudge(t *testing.T) {
	cases := []struct {
		a, b game.Choice
		want int
	}{{game.Rock, game.Scissors, 1}, {game.Scissors, game.Paper, 1}, {game.Paper, game.Rock, 1}, {game.Scissors, game.Rock, -1}, {game.Paper, game.Scissors, -1}, {game.Rock, game.Paper, -1}, {game.Rock, game.Rock, 0}, {game.Paper, game.Paper, 0}, {game.Scissors, game.Scissors, 0}}
	for _, tc := range cases {
		t.Run(string(tc.a)+"_"+string(tc.b), func(t *testing.T) {
			if got := game.Judge(tc.a, tc.b); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}
func TestValidChoice(t *testing.T) {
	for _, c := range []game.Choice{"", "lizard", "ROCK"} {
		if game.ValidChoice(c) {
			t.Fatalf("accepted %q", c)
		}
	}
}
