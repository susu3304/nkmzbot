package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
	"github.com/susu3304/nkmzbot/internal/imm"
)

func HandleImm(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner) {
	data := i.ApplicationCommandData()
	if len(data.Options) == 0 {
		respondText(s, i, "サブコマンドが指定されていません。")
		return
	}

	top := data.Options[0]
	switch top.Name {
	case "run":
		if runner == nil {
			respondText(s, i, "IMM runner is not configured.")
			return
		}
		source := stringOptionValue(top.Options, "source")
		rawArgs := stringOptionValue(top.Options, "args")
		runImmInteraction(s, i, cli, runner, source, rawArgs, false)
	case "check":
		if runner == nil {
			respondText(s, i, "IMM runner is not configured.")
			return
		}
		source := stringOptionValue(top.Options, "source")
		checkImmInteraction(s, i, cli, runner, source)
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
	message := selectedResolvedMessage(data)
	if message == nil {
		respondText(s, i, "メッセージが見つかりませんでした。")
		return
	}
	if strings.TrimSpace(message.Content) == "" {
		respondText(s, i, "IMMとして実行できる本文がありません。")
		return
	}

	showRunMessageAsIMMModal(s, i, message.ID)
}

func HandleRegisterMessageAsIMM(s *discordgo.Session, i *discordgo.InteractionCreate, runner *imm.Runner) {
	if runner == nil {
		respondText(s, i, "IMM runner is not configured.")
		return
	}
	data := i.ApplicationCommandData()
	message := selectedResolvedMessage(data)
	if message == nil {
		respondText(s, i, "メッセージが見つかりませんでした。")
		return
	}
	if strings.TrimSpace(message.Content) == "" {
		respondText(s, i, "IMMコマンドとして登録できる本文がありません。")
		return
	}

	showRegisterMessageAsIMMModal(s, i, message.ID)
}

func showRunMessageAsIMMModal(s *discordgo.Session, i *discordgo.InteractionCreate, messageID string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalRunMessageAsIMMPrefix + messageID,
			Title:    "IMMを実行",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "imm_args",
							Label:       "引数",
							Style:       discordgo.TextInputShort,
							Placeholder: `例: a "hello world"`,
							Required:    false,
							MaxLength:   1000,
						},
					},
				},
			},
		},
	})
}

func showRegisterMessageAsIMMModal(s *discordgo.Session, i *discordgo.InteractionCreate, messageID string) {
	_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseModal,
		Data: &discordgo.InteractionResponseData{
			CustomID: modalRegisterMessageIMMPrefix + messageID,
			Title:    "IMMコマンドを登録",
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "command_name",
							Label:       "コマンド名",
							Style:       discordgo.TextInputShort,
							Placeholder: "例: repeat",
							Required:    true,
							MaxLength:   50,
						},
					},
				},
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.TextInput{
							CustomID:    "tags",
							Label:       "タグ",
							Style:       discordgo.TextInputShort,
							Placeholder: "例: imm utility",
							Required:    false,
							MaxLength:   200,
						},
					},
				},
			},
		},
	})
}

func HandleRunMessageAsIMMModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, data discordgo.ModalSubmitInteractionData) {
	if runner == nil {
		respondText(s, i, "IMM runner is not configured.")
		return
	}
	messageID := strings.TrimPrefix(data.CustomID, modalRunMessageAsIMMPrefix)
	message, err := s.ChannelMessage(i.ChannelID, messageID)
	if err != nil {
		respondText(s, i, "メッセージの取得に失敗しました。")
		return
	}
	if strings.TrimSpace(message.Content) == "" {
		respondText(s, i, "IMMとして実行できる本文がありません。")
		return
	}

	runImmInteraction(s, i, cli, runner, message.Content, modalInputValue(data, "imm_args"), false)
}

func HandleRegisterMessageAsIMMModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, data discordgo.ModalSubmitInteractionData) {
	if runner == nil {
		respondText(s, i, "IMM runner is not configured.")
		return
	}
	name := normalizeImmCommandName(modalInputValue(data, "command_name"))
	if name == "" {
		respondText(s, i, "コマンド名が入力されていません。")
		return
	}

	messageID := strings.TrimPrefix(data.CustomID, modalRegisterMessageIMMPrefix)
	message, err := s.ChannelMessage(i.ChannelID, messageID)
	if err != nil {
		respondText(s, i, "メッセージの取得に失敗しました。")
		return
	}
	source := message.Content
	if strings.TrimSpace(source) == "" {
		respondText(s, i, "IMMコマンドとして登録できる本文がありません。")
		return
	}

	if !deferInteraction(s, i) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), CommandExpansionTimeout(runner))
	defer cancel()
	expanded, err := NewCommandExpander(cli, runner).ExpandRequest(ctx, interactionExecutionContext(i), source, "", nil)
	if err != nil {
		editInteraction(s, i, "コマンド展開に失敗しました: "+err.Error())
		return
	}
	check, err := runner.Check(ctx, immRequest(i, expanded.Source, expanded.RawArgs, expanded.Args))
	if err != nil {
		editInteraction(s, i, "IMMチェックに失敗しました: "+err.Error())
		return
	}
	if check.ExitCode != 0 || check.TimedOut || check.OutputTruncated {
		editInteraction(s, i, FormatImmFailure("IMMチェックに失敗しました", check))
		return
	}

	if err := cli.AddCommandWithKind(i.GuildID, name, source, "imm", parseTags(modalInputValue(data, "tags"))...); err != nil {
		if client.IsConnectionError(err) {
			editInteraction(s, i, apiConnectionErrorMessage)
		} else {
			editInteraction(s, i, "IMMコマンドの保存に失敗しました: "+err.Error())
		}
		return
	}
	editInteraction(s, i, fmt.Sprintf("IMMコマンド `?%s` を追加しました。", name))
}

func handleImmCommandGroup(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, group *discordgo.ApplicationCommandInteractionDataOption) {
	if len(group.Options) == 0 {
		respondText(s, i, "IMM commandのサブコマンドが指定されていません。")
		return
	}

	sub := group.Options[0]
	switch sub.Name {
	case "add", "update":
		if runner == nil {
			respondText(s, i, "IMM runner is not configured.")
			return
		}
		name := normalizeImmCommandName(stringOptionValue(sub.Options, "name"))
		source := stringOptionValue(sub.Options, "source")
		tags := parseTags(stringOptionValue(sub.Options, "tags"))
		if name == "" || strings.TrimSpace(source) == "" {
			respondText(s, i, "nameとsourceが必要です。")
			return
		}

		if !deferInteraction(s, i) {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), CommandExpansionTimeout(runner))
		defer cancel()
		expanded, err := NewCommandExpander(cli, runner).ExpandRequest(ctx, interactionExecutionContext(i), source, "", nil)
		if err != nil {
			editInteraction(s, i, "コマンド展開に失敗しました: "+err.Error())
			return
		}
		check, err := runner.Check(ctx, immRequest(i, expanded.Source, expanded.RawArgs, expanded.Args))
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
		editInteraction(s, i, fmt.Sprintf("IMMコマンド `?%s` を%sしました。", name, action))
	case "remove":
		name := normalizeImmCommandName(stringOptionValue(sub.Options, "name"))
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
		respondText(s, i, fmt.Sprintf("IMMコマンド `?%s` を削除しました。", name))
	case "show":
		name := normalizeImmCommandName(stringOptionValue(sub.Options, "name"))
		if name == "" {
			respondText(s, i, "nameが必要です。")
			return
		}
		cmd, err := cli.GetCommand(i.GuildID, name)
		if err != nil {
			if client.IsConnectionError(err) {
				respondText(s, i, apiConnectionErrorMessage)
			} else {
				respondText(s, i, "IMMコマンドの取得に失敗しました。")
			}
			return
		}
		if cmd == nil || !strings.EqualFold(commandKind(cmd.Kind), "imm") {
			respondText(s, i, fmt.Sprintf("IMMコマンド `?%s` は見つかりません。", name))
			return
		}
		respondText(s, i, formatImmCommandSource(name, cmd.Response))
	default:
		respondText(s, i, "未知のIMM commandサブコマンドです。")
	}
}

func normalizeImmCommandName(name string) string {
	return strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(name), "!?"))
}

func runImmInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, source, rawArgs string, trace bool) {
	args, err := imm.SplitArgs(rawArgs)
	if err != nil {
		respondText(s, i, "argsの解釈に失敗しました: "+err.Error())
		return
	}
	if !deferInteraction(s, i) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), CommandExpansionTimeout(runner))
	defer cancel()
	expanded, err := NewCommandExpander(cli, runner).ExpandRequest(ctx, interactionExecutionContext(i), source, rawArgs, args)
	if err != nil {
		editInteraction(s, i, "コマンド展開に失敗しました: "+err.Error())
		return
	}
	req := immRequest(i, expanded.Source, expanded.RawArgs, expanded.Args)
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

func checkImmInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, cli *client.Client, runner *imm.Runner, source string) {
	if !deferInteraction(s, i) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), CommandExpansionTimeout(runner))
	defer cancel()
	expanded, err := NewCommandExpander(cli, runner).ExpandRequest(ctx, interactionExecutionContext(i), source, "", nil)
	if err != nil {
		editInteraction(s, i, "コマンド展開に失敗しました: "+err.Error())
		return
	}
	result, err := runner.Check(ctx, immRequest(i, expanded.Source, expanded.RawArgs, expanded.Args))
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

func NewCommandExpander(cli *client.Client, runner *imm.Runner, stack ...string) CommandExpander {
	return CommandExpander{
		Client:   cli,
		Runner:   runner,
		MaxDepth: DefaultCommandExpansionLimit,
		Stack:    stack,
	}
}

func interactionExecutionContext(i *discordgo.InteractionCreate) CommandExecutionContext {
	userID := ""
	if i.Member != nil && i.Member.User != nil {
		userID = i.Member.User.ID
	} else if i.User != nil {
		userID = i.User.ID
	}
	return CommandExecutionContext{
		GuildID:   i.GuildID,
		ChannelID: i.ChannelID,
		UserID:    userID,
	}
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

func formatImmCommandSource(name, source string) string {
	name = normalizeImmCommandName(name)
	source = strings.TrimRight(source, "\r\n")
	if source == "" {
		source = "(empty)"
	}
	header := fmt.Sprintf("IMMコマンド `?%s` の中身:\n", name)
	return formatBoundedCodeBlock(header, "imm", source)
}

func formatBoundedCodeBlock(header, language, content string) string {
	content = escapeDiscordCodeFence(content)
	prefix := header + "```" + language + "\n"
	suffix := "\n```"
	available := discordMessageLimit - runeLen(prefix) - runeLen(suffix)
	if available <= 0 {
		return truncateDiscordMessage(header)
	}
	if runeLen(content) <= available {
		return prefix + content + suffix
	}
	notice := "\n...(truncated)"
	available -= runeLen(notice)
	if available < 0 {
		available = 0
	}
	return prefix + firstRunes(content, available) + notice + suffix
}

func truncateDiscordMessage(content string) string {
	if runeLen(content) <= discordMessageLimit {
		return content
	}
	notice := "\n...(truncated)"
	available := discordMessageLimit - runeLen(notice)
	if available < 0 {
		available = 0
	}
	return firstRunes(content, available) + notice
}

func truncateForCodeBlock(content string) string {
	content = escapeDiscordCodeFence(content)
	return truncateDiscordMessage(content)
}

const discordMessageLimit = 2000

func escapeDiscordCodeFence(content string) string {
	return strings.ReplaceAll(content, "```", "`\u200b``")
}

func runeLen(content string) int {
	return len([]rune(content))
}

func firstRunes(content string, limit int) string {
	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return string(runes[:limit])
}
