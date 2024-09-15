package main

import (
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

func sendTelegramMessage(title string, upperParameters, lowerParameters map[string]Parameters) {
	bot, err := tgbotapi.NewBotAPI(BOT_TOKEN)
	if err != nil {
		fmt.Println(err)
	}

	lengthUptrend := len(upperParameters)
	lengthDowntrend := len(lowerParameters)

	message := fmt.Sprintf("["+title+"]"+"\n\n ✅ Got %v coins alert uptrend!", lengthUptrend)
	if lengthUptrend > 0 {
		for coin, _ := range upperParameters {
			message += "\n" + coin
		}
	}

	message = fmt.Sprintf(message+"\n\n ⛔ Got %v coins alert downtrend!", lengthDowntrend)
	if lengthDowntrend > 0 {
		for coin, _ := range lowerParameters {
			message += "\n" + coin
		}
	}

	msg := tgbotapi.NewMessage(RECEIVER_USER_ID, message)
	_, err = bot.Send(msg)
	if err != nil {
		fmt.Println(err)
	}
}
