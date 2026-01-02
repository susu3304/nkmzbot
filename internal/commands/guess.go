package commands

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/geourl"
	"github.com/susu3304/nkmzbot/internal/guess"
)

type GuessCommand struct {
	Svc *guess.Service
}

func (c *GuessCommand) Def() *discordgo.ApplicationCommand {
	return &discordgo.ApplicationCommand{
		Name:         "guess",
		Description:  "ジオゲッサーを開始・プレイします",
		DMPermission: boolPtr(false),
		Options: []*discordgo.ApplicationCommandOption{
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "start",
				Description: "このチャンネルでジオゲッサーセッションを開始",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "stop",
				Description: "このチャンネルのセッションを終了",
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "guess",
				Description: "推測を送信",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "url",
						Description: "Google Mapsの短縮URL",
						Required:    true,
					},
				},
			},
			{
				Type:        discordgo.ApplicationCommandOptionSubCommand,
				Name:        "answer",
				Description: "正解を発表してスコアを表示",
				Options: []*discordgo.ApplicationCommandOption{
					{
						Type:        discordgo.ApplicationCommandOptionString,
						Name:        "url",
						Description: "正解のGoogle Maps URL",
						Required:    true,
					},
				},
			},
		},
	}
}

func (c *GuessCommand) Handler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		respondText(s, i, "サブコマンドが指定されていません")
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
		err := c.Svc.StartSession(context.Background(), channelID, gid, userID)
		if err != nil {
			if err == guess.ErrSessionAlreadyExists {
				respondText(s, i, "このチャンネルには既にセッションが開始されています")
			} else {
				respondText(s, i, "セッションの開始に失敗しました: "+err.Error())
			}
			return
		}
		respondText(s, i, "✅ ジオゲッサーセッションを開始しました！\n`/guess <Google Maps URL>` で推測を送信してください")

	case "stop":
		err := c.Svc.StopSession(context.Background(), channelID)
		if err != nil {
			if err == guess.ErrNoActiveSession {
				respondText(s, i, "このチャンネルにはアクティブなセッションがありません")
			} else {
				respondText(s, i, "セッションの終了に失敗しました: "+err.Error())
			}
			return
		}
		respondText(s, i, "✅ セッションを終了しました")

	case "guess":
		urlOpt := getStringOption(sub.Options, "url")
		if urlOpt == nil {
			respondText(s, i, "URLの指定が必要です")
			return
		}

		// First, defer the response since URL expansion might take time
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return
		}

		// Expand URL and extract coordinates
		lat, lng, finalURL, err := geourl.ExpandAndExtractCoords(*urlOpt)
		if err != nil {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: strPtr("座標の抽出に失敗しました: " + err.Error()),
			})
			return
		}

		// Add guess to session
		err = c.Svc.AddGuess(context.Background(), channelID, userID, lat, lng, finalURL)
		if err != nil {
			if err == guess.ErrNoActiveSession {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("このチャンネルにはアクティブなセッションがありません\n`/guess start` でセッションを開始してください"),
				})
			} else {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("推測の記録に失敗しました: " + err.Error()),
				})
			}
			return
		}

		msg := fmt.Sprintf("✅ <@%s> の推測を記録しました！", userID)
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})

	case "answer":
		urlOpt := getStringOption(sub.Options, "url")
		if urlOpt == nil {
			respondText(s, i, "URLの指定が必要です")
			return
		}

		// First, defer the response since URL expansion might take time
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		})
		if err != nil {
			return
		}

		// Expand URL and extract coordinates
		lat, lng, finalURL, err := geourl.ExpandAndExtractCoords(*urlOpt)
		if err != nil {
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: strPtr("座標の抽出に失敗しました: " + err.Error()),
			})
			return
		}

		// Set answer and calculate scores
		results, err := c.Svc.SetAnswer(context.Background(), channelID, lat, lng, finalURL)
		if err != nil {
			if err == guess.ErrNoActiveSession {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("このチャンネルにはアクティブなセッションがありません"),
				})
			} else {
				s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
					Content: strPtr("スコアの計算に失敗しました: " + err.Error()),
				})
			}
			return
		}

		if len(results) == 0 {
			msg := fmt.Sprintf("📍 正解: %s\n\nまだ誰も推測していません", finalURL)
			s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
				Content: &msg,
			})
			return
		}

		// Sort results by score (descending)
		sort.Slice(results, func(i, j int) bool {
			return results[i].Score > results[j].Score
		})

		var b strings.Builder
		fmt.Fprintf(&b, "📍 **正解**: %s\n\n", finalURL)
		fmt.Fprintf(&b, "🏆 **結果** (%d名)\n", len(results))
		fmt.Fprintf(&b, "```\n")
		for idx, r := range results {
			rank := idx + 1
			emoji := ""
			if rank == 1 {
				emoji = "🥇"
			} else if rank == 2 {
				emoji = "🥈"
			} else if rank == 3 {
				emoji = "🥉"
			}
			fmt.Fprintf(&b, "%s %d位: %5d点 (%s)\n", emoji, rank, r.Score, guess.FormatDistance(r.DistanceMeters))
		}
		fmt.Fprintf(&b, "```\n")
		for idx, r := range results {
			rank := idx + 1
			fmt.Fprintf(&b, "%d. <@%s>: **%d点** (距離: %s)\n   推測: %s\n", rank, r.UserID, r.Score, guess.FormatDistance(r.DistanceMeters), r.GuessURL)
		}

		msg := b.String()
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})

	default:
		respondText(s, i, "未知のサブコマンドです")
	}
}

func strPtr(s string) *string {
	return &s
}
