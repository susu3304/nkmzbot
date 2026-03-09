package commands

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
)

func HandleWallet(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		respondText(s, i, "サブコマンドが指定されていません")
		return
	}
	if i.Member == nil || i.Member.User == nil {
		respondText(s, i, "実行ユーザーを識別できませんでした")
		return
	}

	actorUserID := i.Member.User.ID
	sub := data.Options[0]
	targetUserID := getUserID(data, sub, "user")
	if targetUserID == "" {
		respondText(s, i, "user の指定が必要です")
		return
	}
	if targetUserID == actorUserID {
		respondText(s, i, "自分自身は指定できません")
		return
	}

	amount := getIntOption(sub.Options, "amount")
	if amount == nil || *amount <= 0 {
		respondText(s, i, "amount は 1 以上の整数で指定してください")
		return
	}

	memo := ""
	if memoOpt := getStringOption(sub.Options, "memo"); memoOpt != nil {
		memo = strings.TrimSpace(*memoOpt)
	}

	switch sub.Name {
	case "pay":
		event, err := cli.CreateWalletTransfer(actorUserID, targetUserID, *amount, memo)
		if err != nil {
			respondText(s, i, walletErrorMessage(err))
			return
		}
		message := fmt.Sprintf("送金を記録しました: <@%s> -> <@%s> %d円", actorUserID, targetUserID, event.Amount)
		if event.Note != "" {
			message += "\nメモ: " + event.Note
		}
		respondText(s, i, message)
	case "req":
		event, err := cli.CreateWalletRequest(actorUserID, targetUserID, *amount, memo)
		if err != nil {
			respondText(s, i, walletErrorMessage(err))
			return
		}
		message := fmt.Sprintf("請求を記録しました: <@%s> -> <@%s> %d円", targetUserID, actorUserID, event.Amount)
		if event.Note != "" {
			message += "\nメモ: " + event.Note
		}
		respondText(s, i, message)
	default:
		respondText(s, i, "未知のサブコマンドです")
	}
}

func walletErrorMessage(err error) string {
	if client.IsConnectionError(err) {
		return apiConnectionErrorMessage
	}

	msg := err.Error()
	switch {
	case strings.Contains(msg, "INVALID_AMOUNT"):
		return "amount は 1 以上の整数で指定してください"
	case strings.Contains(msg, "SELF_TRANSFER_NOT_ALLOWED"):
		return "自分自身との送金・請求はできません"
	case strings.Contains(msg, "COUNTERPARTY_NOT_FOUND"):
		return "相手ユーザーが見つかりません"
	case strings.Contains(msg, "WALLET_FROZEN"):
		return "Wallet が凍結されているため操作できません"
	case strings.Contains(msg, "UNAUTHORIZED"):
		return "Wallet API で認証できませんでした"
	default:
		return "Wallet の記録に失敗しました: " + msg
	}
}
