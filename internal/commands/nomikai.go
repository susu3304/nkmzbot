package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/db"
	"github.com/susu3304/nkmzbot/internal/nomikai"
)

type NomikaiCommand struct {
	Svc *nomikai.Service
	DB  *db.DB
}

func (c *NomikaiCommand) Def() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:         "nomikai",
		Description:  "飲み会割り勘セッションを操作します",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "start",
				Description: "このチャンネルでセッションを開始",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "stop",
				Description: "このチャンネルのセッションを終了",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "join",
				Description: "自分を参加者に追加",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "member",
				Description: "指定ユーザーを参加者に追加",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "users",
						Description: "追加するユーザー（メンション/IDをスペース区切り。単一も可）",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "weight",
				Description: "参加者の比率を設定",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "users",
						Description: "対象ユーザー（メンション/IDをスペース区切り。単一も可）",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionNumber,
						Name:        "value",
						Description: "比率 (例: 1.5)",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "tatekae",
				Description: "立替（支出）を記録（負額も可）",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionInteger,
						Name:        "amount",
						Description: "金額（円）",
						Required:    true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "payer",
						Description: "支払者（未指定なら自分）",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "for",
						Description: "対象ユーザー（メンション/ID。スペース区切りで複数可）",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "memo",
						Description: "メモ",
						Required:    false,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "settle",
				Description: "ネット精算を計算",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "status",
				Description: "現在の状況を表示",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "memberlist",
				Description: "参加中のメンバーを表示",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "remind",
				Description: "未払いタスクの定期リマインドを設定し即時送信",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "interval",
						Description: "リマインド間隔 (例: 1d2h3m / デフォルト1d / 最小1m)",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "state",
						Description: "on/off (on=有効, off=停止)",
						Required:    false,
						Choices: []*discordgo.ApplicationCommandOptionChoice{
							{Name: "on", Value: "on"},
							{Name: "off", Value: "off"},
						},
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "seisan",
				Description: "精算の支払いを登録して未払いを減らす",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "to",
						Description: "受け取り側",
						Required:    true,
					},
					{
						Type:         discordgo.ApplicationCommandOptionString,
						Name:         "amount",
						Description:  "支払った金額 (円) / all=未払い全額",
						Required:     true,
						Autocomplete: true,
					},
					{
						Type:        discordgo.ApplicationCommandOptionUser,
						Name:        "payer",
						Description: "支払者 (未指定なら自分)",
						Required:    false,
					},
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "memo",
						Description: "メモ",
						Required:    false,
					},
				},
			},
		},
	}
}

func (c *NomikaiCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{Content: "サブコマンドが指定されていません"},
		})
		return
	}

	sub := data.Options[0]
	channelID := i.ChannelID
	userID := i.Member.User.ID

	switch sub.Name {
	case "start":
		// Parse guild ID to int64
		gid, errParse := strconv.ParseInt(i.GuildID, 10, 64)
		if errParse != nil || gid == 0 {
			respondText(s, i, "ギルドIDの取得に失敗しました")
			return
		}
		// Defaults: rounding=1, remainder strategy="organizer"
		err := c.Svc.StartSession(context.Background(), channelID, gid, userID, 1, "organizer")
		respondSimple(s, i, err, "このチャンネルでセッションを開始しました", "既に開始されています")
	case "stop":
		err := c.Svc.StopSession(context.Background(), channelID)
		respondSimple(s, i, err, "セッションを終了しました", "セッションが存在しません")
	case "join":
		err := c.Svc.Join(context.Background(), channelID, userID)
		respondSimple(s, i, err, "参加者として登録しました", "セッションが開始されていません")
	case "member":
		usersOpt := getStringOption(sub.Options, "users")
		if usersOpt == nil {
			respondText(s, i, "users の指定が必要です")
			return
		}
		ids := ParseMentionIDs(*usersOpt)
		if len(ids) == 0 {
			respondText(s, i, "ユーザーのメンション/IDを認識できませんでした")
			return
		}
		for _, id := range ids {
			if err := c.Svc.Join(context.Background(), channelID, id); err != nil {
				respondText(s, i, "セッションが開始されていません")
				return
			}
		}
		if len(ids) == 1 {
			respondText(s, i, fmt.Sprintf("<@%s> を参加者に追加しました", ids[0]))
		} else {
			var b strings.Builder
			fmt.Fprintf(&b, "%d 名を参加者に追加しました\n追加: ", len(ids))
			for idx, id := range ids {
				if idx > 0 {
					b.WriteString(", ")
				}
				fmt.Fprintf(&b, "<@%s>", id)
			}
			respondText(s, i, b.String())
		}
	case "weight":
		usersOpt := getStringOption(sub.Options, "users")
		val := getNumberOption(sub.Options, "value")
		if usersOpt == nil || val == nil {
			respondText(s, i, "users と value の指定が必要です")
			return
		}
		ids := ParseMentionIDs(*usersOpt)
		if len(ids) == 0 {
			respondText(s, i, "ユーザーのメンション/IDを認識できませんでした")
			return
		}
		var joinedIDs []string
		for _, id := range ids {
			joined, _ := c.Svc.SetWeight(context.Background(), channelID, id, *val)
			if joined {
				joinedIDs = append(joinedIDs, id)
			}
		}
		if len(ids) == 1 {
			msg := fmt.Sprintf("<@%s> の比率を %.2f に設定しました", ids[0], *val)
			if len(joinedIDs) == 1 {
				msg += "\nこのユーザーを参加登録しました"
			}
			respondText(s, i, msg)
		} else {
			msg := fmt.Sprintf("%d 名の比率を %.2f に設定しました", len(ids), *val)
			if len(joinedIDs) > 0 {
				msg += "\n参加登録: "
				for idx, id := range joinedIDs {
					if idx > 0 {
						msg += ", "
					}
					msg += fmt.Sprintf("<@%s>", id)
				}
			}
			respondText(s, i, msg)
		}
	case "tatekae":
		amtOpt := getIntOption(sub.Options, "amount")
		memoOpt := getStringOption(sub.Options, "memo")
		forOpt := getStringOption(sub.Options, "for")
		if amtOpt == nil {
			respondText(s, i, "金額の指定が必要です")
			return
		}
		memo := ""
		if memoOpt != nil {
			memo = *memoOpt
		}
		payerID := getUserID(data, sub, "payer")
		payer := userID
		if payerID != "" {
			payer = payerID
		}
		var beneficiaries []string
		if forOpt != nil {
			beneficiaries = ParseMentionIDs(*forOpt)
		}
		var joined bool
		var benJoined []string
		var err error
		if len(beneficiaries) > 0 {
			joined, benJoined, err = c.Svc.AddPaymentFor(context.Background(), channelID, payer, *amtOpt, memo, beneficiaries)
		} else {
			joined, err = c.Svc.AddPayment(context.Background(), channelID, payer, *amtOpt, memo)
		}
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		msg := fmt.Sprintf("<@%s> の支払として %d 円を記録しました", payer, *amtOpt)
		if len(beneficiaries) > 0 {
			msg += "\n対象: "
			for idx, id := range beneficiaries {
				if idx > 0 {
					msg += ", "
				}
				msg += fmt.Sprintf("<@%s>", id)
			}
		}
		// Compose join notifications (payer and newly joined beneficiaries)
		var joinIDs []string
		if joined {
			joinIDs = append(joinIDs, payer)
		}
		if len(benJoined) > 0 {
			joinIDs = append(joinIDs, benJoined...)
		}
		if len(joinIDs) > 0 {
			msg += "\n参加登録: "
			for idx, id := range joinIDs {
				if idx > 0 {
					msg += ", "
				}
				msg += fmt.Sprintf("<@%s>", id)
			}
		}
		respondText(s, i, msg)
	case "settle":
		res, err := c.Svc.Settle(context.Background(), channelID)
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		if len(res.Tasks) == 0 {
			respondText(s, i, "精算は不要です")
			return
		}
		respondText(s, i, res.Summary)
	case "status":
		txt, err := c.Svc.Status(context.Background(), channelID)
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		respondText(s, i, txt)
	case "memberlist":
		ids, err := c.Svc.Members(context.Background(), channelID)
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		var b strings.Builder
		fmt.Fprintf(&b, "参加者 (%d名):\n", len(ids))
		for _, id := range ids {
			fmt.Fprintf(&b, "・<@%s>\n", id)
		}
		respondText(s, i, b.String())
	case "remind":
		intervalMinutes := 0
		if opt := getStringOption(sub.Options, "interval"); opt != nil {
			mins, err := ParseDHMToMinutes(*opt)
			if err != nil {
				respondText(s, i, err.Error())
				return
			}
			intervalMinutes = mins
		}
		disable := false
		if opt := getStringOption(sub.Options, "state"); opt != nil {
			state := strings.ToLower(strings.TrimSpace(*opt))
			switch state {
			case "on", "enable":
				disable = false
			case "off", "disable":
				disable = true
			case "オン":
				disable = false
			case "オフ":
				disable = true
			default:
				respondText(s, i, "state は on/off で指定してください")
				return
			}
		}
		msg, err := c.Svc.ConfigureReminder(context.Background(), channelID, intervalMinutes, disable, true)
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		respondText(s, i, msg)
	case "seisan":
		amtStrOpt := getStringOption(sub.Options, "amount")
		if amtStrOpt == nil {
			respondText(s, i, "amount の指定が必要です")
			return
		}
		payee := getUserID(data, sub, "to")
		if payee == "" {
			respondText(s, i, "to の指定が必要です")
			return
		}
		payer := getUserID(data, sub, "payer")
		if payer == "" {
			payer = userID
		}
		memo := ""
		if opt := getStringOption(sub.Options, "memo"); opt != nil {
			memo = *opt
		}

		amountStr := strings.TrimSpace(*amtStrOpt)
		payAll := false
		amount := int64(0)
		switch strings.ToLower(amountStr) {
		case "all", "zenbu", "全部", "全額":
			payAll = true
		default:
			v, err := strconv.ParseInt(amountStr, 10, 64)
			if err != nil || v <= 0 {
				respondText(s, i, "amount は正の数、または all を指定してください")
				return
			}
			amount = v
		}

		msg, err := c.Svc.RegisterPayment(context.Background(), channelID, payer, payee, amount, memo, userID, payAll)
		if err != nil {
			respondText(s, i, err.Error())
			return
		}
		respondText(s, i, msg)
	default:
		respondText(s, i, "未知のサブコマンドです")
	}
}

func (c *NomikaiCommand) AutocompleteHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	if data.Name != "nomikai" {
		return
	}
	if len(data.Options) == 0 {
		return
	}
	sub := data.Options[0]
	if sub.Name != "seisan" {
		return
	}

	// Find focused option in the subcommand.
	focusedName := ""
	userInput := ""
	for _, opt := range sub.Options {
		if opt.Focused {
			focusedName = opt.Name
			userInput = opt.StringValue()
			break
		}
	}
	if focusedName != "amount" {
		return
	}

	payeeID := ""
	payerID := ""
	for _, opt := range sub.Options {
		switch opt.Name {
		case "to":
			if id, ok := opt.Value.(string); ok {
				payeeID = id
			}
		case "payer":
			if id, ok := opt.Value.(string); ok {
				payerID = id
			}
		}
	}
	if payerID == "" && i.Member != nil && i.Member.User != nil {
		payerID = i.Member.User.ID
	}

	choices := []*discordgo.ApplicationCommandOptionChoice{
		{Name: "all（未払い全額）", Value: "all"},
	}

	// If we can compute outstanding amount for the pair, also offer it as a one-click numeric choice.
	if payerID != "" && payeeID != "" {
		ev, err := c.DB.ActiveEventByChannel(context.Background(), i.ChannelID)
		if err == nil && ev != nil {
			out, err := c.DB.OutstandingSettlementAmount(context.Background(), ev.ID, payerID, payeeID)
			if err == nil && out > 0 {
				choices = append([]*discordgo.ApplicationCommandOptionChoice{
					{Name: fmt.Sprintf("%d（未払い全額）", out), Value: strconv.FormatInt(out, 10)},
				}, choices...)
			}
		}
	}

	// If user typed something, echo it as a choice so they can commit it quickly.
	if strings.TrimSpace(userInput) != "" {
		choices = append(choices, &discordgo.ApplicationCommandOptionChoice{Name: userInput, Value: userInput})
	}
	if len(choices) > 25 {
		choices = choices[:25]
	}

	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionApplicationCommandAutocompleteResult,
		Data: &discordgo.InteractionResponseData{Choices: choices},
	})
}

func respondSimple(s *discordgo.Session, i *discordgo.InteractionCreate, err error, ok, ng string) {
	if err != nil {
		respondText(s, i, ng)
		return
	}
	respondText(s, i, ok)
}

func respondText(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Content: content},
	})
}

func getUserID(data discordgo.ApplicationCommandInteractionData, sub *discordgo.ApplicationCommandInteractionDataOption, name string) string {
	for _, o := range sub.Options {
		if o.Name != name {
			continue
		}
		// Prefer raw ID from option value
		if id, ok := o.Value.(string); ok && id != "" {
			return id
		}
		// Fallback to resolved user (if available)
		if data.Resolved != nil {
			// When only one user is resolved and this option targets a user, return its ID
			for id := range data.Resolved.Users {
				return id
			}
		}
		// Last resort: try UserValue (may require session; nil is tolerated)
		if u := o.UserValue(nil); u != nil {
			return u.ID
		}
	}
	return ""
}

func getNumberOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *float64 {
	for _, o := range opts {
		if o.Name == name {
			v := o.FloatValue()
			return &v
		}
	}
	return nil
}

func getIntOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *int64 {
	for _, o := range opts {
		if o.Name == name {
			v := o.IntValue()
			return &v
		}
	}
	return nil
}

func getStringOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *string {
	for _, o := range opts {
		if o.Name == name {
			v := o.StringValue()
			return &v
		}
	}
	return nil
}

func getBoolOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *bool {
	for _, o := range opts {
		if o.Name == name {
			v := o.BoolValue()
			return &v
		}
	}
	return nil
}

// no session needed for reading raw ID from options
