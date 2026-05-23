package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
	"github.com/susu3304/nkmzbot/internal/imm"
)

func HandleImm(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner) {
	if runner == nil {
		respondText(s, i, "IMM runner is not configured.")
		return
	}
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		respondText(s, i, "サブコマンドが指定されていません。")
		return
	}

	top := data.Options[0]
	switch top.Name {
	case "run":
		source := stringOptionValue(top.Options, "source")
		rawArgs := stringOptionValue(top.Options, "args")
		runImmInteraction(s, i, runner, source, rawArgs, false)
	case "check":
		source := stringOptionValue(top.Options, "source")
		checkImmInteraction(s, i, runner, source)
	case "command":
		handleImmCommandGroup(s, i, cli, runner, top)
	default:
		respondText(s, i, "未知のIMMサブコマンドです。")
	}
}

func HandleRunMessageAsIMM(s *discordgo.Session, i *discordgo.InteractionCreate, runner *imm.Runner) {
	if runner == nil {
		respondText(s, i, "IMM runner is not configured.")
		return
	}
	data := i.ApplicationCommandData()
	if data.Resolved == nil || len(data.Resolved.Messages) == 0 {
		respondText(s, i, "メッセージが見つかりませんでした。")
		return
	}

	var message *discordgo.Message
	for _, msg := range data.Resolved.Messages {
		message = msg
		break
	}
	if message == nil || strings.TrimSpace(message.Content) == "" {
		respondText(s, i, "IMMとして実行できる本文がありません。")
		return
	}

	runImmInteraction(s, i, runner, message.Content, "", false)
}

func handleImmCommandGroup(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, group *discordgo.ApplicationCommandInteractionDataOption) {
	if len(group.Options) == 0 {
		respondText(s, i, "IMM commandのサブコマンドが指定されていません。")
		return
	}

	sub := group.Options[0]
	switch sub.Name {
	case "add", "update":
		name := strings.TrimPrefix(strings.TrimSpace(stringOptionValue(sub.Options, "name")), "!")
		source := stringOptionValue(sub.Options, "source")
		tags := parseTags(stringOptionValue(sub.Options, "tags"))
		if name == "" || strings.TrimSpace(source) == "" {
			respondText(s, i, "nameとsourceが必要です。")
			return
		}

		if !deferInteraction(s, i) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), runner.Timeout+time.Second)
		defer cancel()
		check, err := runner.Check(ctx, immRequest(i, source, "", nil))
		if err != nil {
			editInteraction(s, i, "IMMチェックに失敗しました: "+err.Error())
			return
		}
		if check.ExitCode != 0 || check.TimedOut || check.OutputTruncated {
			editInteraction(s, i, FormatImmFailure("IMMチェックに失敗しました", check))
			return
		}

		var saveErr error
		if sub.Name == "add" {
			saveErr = cli.AddCommandWithKind(i.GuildID, name, source, "imm", tags...)
		} else {
			saveErr = cli.UpdateCommandWithKind(i.GuildID, name, source, "imm", tags...)
		}
		if saveErr != nil {
			if client.IsConnectionError(saveErr) {
				editInteraction(s, i, apiConnectionErrorMessage)
			} else {
				editInteraction(s, i, "IMMコマンドの保存に失敗しました: "+saveErr.Error())
			}
			return
		}
		action := "追加"
		if sub.Name == "update" {
			action = "更新"
		}
		editInteraction(s, i, fmt.Sprintf("IMMコマンド `!%s` を%sしました。", name, action))
	case "remove":
		name := strings.TrimPrefix(strings.TrimSpace(stringOptionValue(sub.Options, "name")), "!")
		if name == "" {
			respondText(s, i, "nameが必要です。")
			return
		}
		err := cli.RemoveCommand(i.GuildID, name)
		if err != nil {
			if client.IsConnectionError(err) {
				respondText(s, i, apiConnectionErrorMessage)
			} else {
				respondText(s, i, "IMMコマンドの削除に失敗しました。")
			}
			return
		}
		respondText(s, i, fmt.Sprintf("IMMコマンド `!%s` を削除しました。", name))
	default:
		respondText(s, i, "未知のIMM commandサブコマンドです。")
	}
}

func runImmInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, runner *imm.Runner, source, rawArgs string, trace bool) {
	args, err := imm.SplitArgs(rawArgs)
	if err != nil {
		respondText(s, i, "argsの解釈に失敗しました: "+err.Error())
		return
	}
	if !deferInteraction(s, i) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), runner.Timeout+time.Second)
	defer cancel()
	req := immRequest(i, source, rawArgs, args)
	req.Trace = trace
	result, err := runner.Run(ctx, req)
	if err != nil {
		editInteraction(s, i, "IMM実行に失敗しました: "+err.Error())
		return
	}
	if result.ExitCode != 0 || result.TimedOut || result.OutputTruncated {
		editInteraction(s, i, FormatImmFailure("IMM実行に失敗しました", result))
		return
	}
	editInteraction(s, i, FormatImmOutput(result.Stdout))
}

func checkImmInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, runner *imm.Runner, source string) {
	if !deferInteraction(s, i) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), runner.Timeout+time.Second)
	defer cancel()
	result, err := runner.Check(ctx, immRequest(i, source, "", nil))
	if err != nil {
		editInteraction(s, i, "IMMチェックに失敗しました: "+err.Error())
		return
	}
	if result.ExitCode != 0 || result.TimedOut || result.OutputTruncated {
		editInteraction(s, i, FormatImmFailure("IMMチェックに失敗しました", result))
		return
	}
	editInteraction(s, i, "IMM check OK")
}

func immRequest(i *discordgo.InteractionCreate, source, rawArgs string, args []string) imm.Request {
	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	return imm.Request{
		Source:    source,
		Args:      args,
		RawArgs:   rawArgs,
		UserID:    userID,
		ChannelID: i.ChannelID,
		GuildID:   i.GuildID,
	}
}

func stringOptionValue(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) string {
	value := getStringOption(opts, name)
	if value == nil {
		return ""
	}
	return *value
}

func deferInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	return err == nil
}

func editInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, content string) {
	content = truncateDiscordMessage(content)
	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{Content: &content})
}

func FormatImmOutput(stdout string) string {
	out := strings.TrimRight(stdout, "\r\n")
	if out == "" {
		return "IMMの出力は空です。"
	}
	return truncateDiscordMessage(out)
}

func FormatImmFailure(prefix string, result imm.Result) string {
	var parts []string
	if result.TimedOut {
		parts = append(parts, "timeout")
	}
	if result.OutputTruncated {
		parts = append(parts, "output truncated")
	}
	if result.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit=%d", result.ExitCode))
	}
	detail := strings.TrimSpace(strings.TrimSpace(result.Stderr) + "\n" + strings.TrimSpace(result.Stdout))
	if detail == "" {
		detail = strings.Join(parts, ", ")
	}
	return truncateDiscordMessage(prefix + "\n```text\n" + truncateForCodeBlock(detail) + "\n```")
}

func truncateDiscordMessage(content string) string {
	const limit = 1900
	if len([]rune(content)) <= limit {
		return content
	}
	runes := []rune(content)
	return string(runes[:limit]) + "\n...(truncated)"
}

func truncateForCodeBlock(content string) string {
	content = strings.ReplaceAll(content, "```", "`\u200b``")
	return truncateDiscordMessage(content)
}
