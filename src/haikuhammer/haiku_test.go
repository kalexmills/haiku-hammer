package haikuhammer

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsHaiku(t *testing.T) {
	haikus := []string {
		"when we're on haikus\nseveral people typing...\nthis discord loves them",
		"bonus points for rhymes,\ncredit for consistent themes,\nHaikus are famous.",
		"\n\nSmall community\nIn a much bigger city \nLives on shitposting",
		"Haikus in English\nCan be enjoyed as well as \nJapanese nightmares",
		"People are single\nPeople don't want to be that\nWe need a big hug",
		"negative space eh,\npositivity you say?\nits all in the mind.",
		"Rules for a Haiku,\nSeven syllables to bridge\nFive syllables; end.",
		"no horny on main\nif we see you post thirst traps\ngo to horny jail",
		"Flowers bloom, Winds change.\nSummer rains, and autumn leaves.\nMount Fuji has snow.",
		"Dearest Mr. Cheese\nPenning his rhyming haiku\nArt license revoked",
		"pegging on their nails,\ninserting cream by the tails.\njust wait for the fails.",
		"\"Go to horny jail\"\nAn exact five syllables\nLet's see what you do.",
		"Hump day is here now\nAutocorrect makes Mem mEm\nThat makes me giggle",
		"Water is vital\nIf y'all don't want to die yet\nBut water is eh",
		"My cat is asleep\nThat bitch woke me up last night\nPayback is a bitch",
		"Never let schooling,\ndisrupt your education.\nquoted from Mark Twain",
		"Poems do not rhyme\nNecessarily at least\nBadger Badger Stop",
		"It is tough enough,\nCrafting Haikus, made to rhyme.\nPoets have it rough.",
		"You rhymed \"rhyme\" with \"rhyme\"\nThat's considered a war crime \nGo to jail, do time",
		"I do have a job,\nI don’t have to go to it,\nBut I get to go.",
		"I will not, cannot\nStop the beauty and grace that\nis known as Haiku",
		"Banana man see\nYou made us haiku crazy\nWhy you do dat now",
		"Another test case\nFor the automated bot\nHere’s a nice haiku",
		"@Alex wrote a bot.\nIt checks for Haiku, so cool.\nReport all bugs please.\n",
		"Hey, um, hey brother?\nDo you know how syllable?\nFive seven five, bitch. :wink:",
	}

	notHaikus := []string{
		"asdf\nsdfg\ngadf",
		"DD lives off smut\nAlso dicks on some dollies\nBut def caffeinated things",
		"it's not a haiku",
		"this\nis\nnot haiku",
		"All haikus  started\nBecause  some bastard decided\nStructure   expected\n \n\t",
		"This is why dogs\nare better than cats they\nwill never sleep",
		"Banana man why\nmake so many haiku\nDo you have job",
	}

	for _, haiku := range haikus {
		assert.Equal(t, true, IsHaiku(haiku), haiku)
	}
	for _, nonHaiku := range notHaikus {
		assert.Equal(t, false, IsHaiku(nonHaiku), nonHaiku)
	}
}