package telegram

import (
	"github.com/AgentGG00/bot-core/src/core/mask"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type Button struct {
	Label string
	Data  string
}

type Update struct {
	ChatID       int64
	Text         string
	CallbackData string
}

type Bot struct {
	api           *tgbotapi.BotAPI
	allowedChatID int64
	masker        *mask.Masker
}

func New(token string, allowedChatID int64, masker *mask.Masker) (*Bot, error) {
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	return &Bot{api: api, allowedChatID: allowedChatID, masker: masker}, nil
}

func (b *Bot) Send(text string) error {
	msg := tgbotapi.NewMessage(b.allowedChatID, b.masker.Mask(text))
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) SendWithButtons(text string, buttons []Button) error {
	row := make([]tgbotapi.InlineKeyboardButton, 0, len(buttons))
	for _, btn := range buttons {
		row = append(row, tgbotapi.NewInlineKeyboardButtonData(btn.Label, btn.Data))
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(row)
	msg := tgbotapi.NewMessage(b.allowedChatID, b.masker.Mask(text))
	msg.ReplyMarkup = keyboard
	_, err := b.api.Send(msg)
	return err
}

func (b *Bot) Updates() <-chan Update {
	out := make(chan Update)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := b.api.GetUpdatesChan(u)

	go func() {
		for upd := range updates {
			if upd.Message != nil {
				if upd.Message.Chat.ID != b.allowedChatID {
					continue
				}
				out <- Update{ChatID: upd.Message.Chat.ID, Text: upd.Message.Text}
			}
			if upd.CallbackQuery != nil {
				if upd.CallbackQuery.Message.Chat.ID != b.allowedChatID {
					continue
				}
				out <- Update{ChatID: upd.CallbackQuery.Message.Chat.ID, CallbackData: upd.CallbackQuery.Data}
				ack := tgbotapi.NewCallback(upd.CallbackQuery.ID, "")
				b.api.Request(ack)
			}
		}
		close(out)
	}()

	return out
}
