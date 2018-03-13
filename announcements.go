package main

func getGameStatus(kills int, deaths int, win bool) string {
	delta := kills - deaths

	// Fuck you go, Build in a math.abs for integers
	if delta < 0 {
		delta = delta * -1
	}

	if delta < 5 && kills < 10 {
		// This game was boring, the player died about as many times as they killed, but overall it was few of both
		if win {
			return "they did okay, and they won"
		} else {
			return "they did okay, but they lost"
		}
	}

	if delta > 5 && kills > deaths {
		// This player killed a shitload of people, and didn't die much
		if win {
			return "they kicked ass and took victory"
		} else {
			return "they couldn't carry hard enough to win"
		}
	}

	if delta > 5 && deaths > kills {
		// This player died a lot
		if win {
			return "they got carried to victory"
		} else {
			return "they got shit on"
		}
	}

	return "here be dragons"
}

