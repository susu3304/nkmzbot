package commands

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/susu3304/nkmzbot/internal/client"
	"github.com/susu3304/nkmzbot/internal/imm"
)

const DefaultCommandExpansionLimit = 3

type CommandExecutionContext struct {
	GuildID   string
	ChannelID string
	UserID    string
}

type CommandExpander struct {
	Client   *client.Client
	Runner   *imm.Runner
	MaxDepth int
	Stack    []string
}

type ExpandedRequest struct {
	Source  string
	Args    []string
	RawArgs string
}

type commandReference struct {
	Prefix string
	Kind   string
	Text   string
}

type commandLookupResult struct {
	Name    string
	Command *client.BotCommandResponse
	RawArgs string
	Args    []string
	Found   bool
}

type expansionArg struct {
	Value      string
	Expandable bool
}

func (e CommandExpander) ExpandRequest(ctx context.Context, execCtx CommandExecutionContext, source, rawArgs string, args []string) (ExpandedRequest, error) {
	return e.expandRequestAtDepth(ctx, execCtx, source, rawArgs, args, 0, append([]string(nil), e.Stack...))
}

func (e CommandExpander) expandRequestAtDepth(ctx context.Context, execCtx CommandExecutionContext, source, rawArgs string, args []string, depth int, stack []string) (ExpandedRequest, error) {
	expandedSource, err := e.expandSource(ctx, execCtx, source, depth, stack)
	if err != nil {
		return ExpandedRequest{}, err
	}
	argTokens, err := expansionArgs(rawArgs, args)
	if err != nil {
		return ExpandedRequest{}, err
	}
	expandedArgs, err := e.expandArgs(ctx, execCtx, argTokens, depth, stack)
	if err != nil {
		return ExpandedRequest{}, err
	}
	if rawArgs != "" || len(args) > 0 {
		rawArgs = joinRawArgs(expandedArgs)
	}
	return ExpandedRequest{Source: expandedSource, Args: expandedArgs, RawArgs: rawArgs}, nil
}

func (e CommandExpander) expandArgs(ctx context.Context, execCtx CommandExecutionContext, args []expansionArg, depth int, stack []string) ([]string, error) {
	expanded := make([]string, 0, len(args))
	for _, arg := range args {
		value := arg.Value
		if !arg.Expandable {
			expanded = append(expanded, value)
			continue
		}
		var err error
		value, err = e.expandValue(ctx, execCtx, value, depth, stack, false)
		if err != nil {
			return nil, err
		}
		expanded = append(expanded, value)
	}
	return expanded, nil
}

func (e CommandExpander) expandValue(ctx context.Context, execCtx CommandExecutionContext, value string, depth int, stack []string, required bool) (string, error) {
	ref, expression, ok := parseExpandableCommandReference(value)
	if !ok {
		return value, nil
	}
	output, found, nextStack, err := e.resolveReference(ctx, execCtx, ref, depth, stack)
	if err != nil {
		return "", err
	}
	if !found {
		if required || expression {
			return "", fmt.Errorf("command reference %q was not found", strings.TrimSpace(value))
		}
		return value, nil
	}
	return e.expandValue(ctx, execCtx, output, depth+1, nextStack, false)
}

func (e CommandExpander) expandSource(ctx context.Context, execCtx CommandExecutionContext, source string, depth int, stack []string) (string, error) {
	var out strings.Builder
	for i := 0; i < len(source); {
		switch {
		case source[i] == '"' || source[i] == '\'':
			next := copyQuoted(&out, source, i)
			i = next
		case source[i] == '#':
			next := copyLineComment(&out, source, i)
			i = next
		case strings.HasPrefix(source[i:], "/*"):
			next := copyBlockComment(&out, source, i)
			i = next
		case hasFunctionNameAt(source, i, "bot_command_body"):
			ref, next, err := parseBotStringCall(source, i, "bot_command_body")
			if err != nil {
				return "", err
			}
			body, err := e.commandBody(execCtx, ref)
			if err != nil {
				return "", err
			}
			out.WriteString(strconv.Quote(body))
			i = next
		case hasFunctionNameAt(source, i, "bot_command_info"):
			ref, next, err := parseBotStringCall(source, i, "bot_command_info")
			if err != nil {
				return "", err
			}
			literal, err := e.commandInfoLiteral(execCtx, ref)
			if err != nil {
				return "", err
			}
			out.WriteString(literal)
			i = next
		case hasFunctionNameAt(source, i, "bot_commands"):
			filter, next, err := parseBotOptionalStringCall(source, i, "bot_commands", "all")
			if err != nil {
				return "", err
			}
			literal, err := e.commandsLiteral(execCtx, filter)
			if err != nil {
				return "", err
			}
			out.WriteString(literal)
			i = next
		case hasFunctionNameAt(source, i, "bot_expand"):
			ref, next, err := parseBotStringCall(source, i, "bot_expand")
			if err != nil {
				return "", err
			}
			output, err := e.expandValue(ctx, execCtx, ref, depth, stack, true)
			if err != nil {
				return "", err
			}
			out.WriteString(strconv.Quote(output))
			i = next
		case hasFunctionNameAt(source, i, "bot_command"):
			ref, next, err := parseBotStringCall(source, i, "bot_command")
			if err != nil {
				return "", err
			}
			output, err := e.expandValue(ctx, execCtx, ref, depth, stack, true)
			if err != nil {
				return "", err
			}
			out.WriteString(strconv.Quote(output))
			i = next
		default:
			out.WriteByte(source[i])
			i++
		}
	}
	return out.String(), nil
}

func (e CommandExpander) resolveReference(ctx context.Context, execCtx CommandExecutionContext, ref commandReference, depth int, stack []string) (string, bool, []string, error) {
	lookup, err := e.lookupReference(execCtx.GuildID, ref)
	if err != nil || !lookup.Found {
		return "", lookup.Found, stack, err
	}
	if depth >= e.maxDepth() {
		key := ref.Prefix + lookup.Name
		return "", true, stack, fmt.Errorf("command expansion depth exceeded: %s", strings.Join(append(stack, key), " -> "))
	}

	key := ref.Prefix + lookup.Name
	nextStack := append(append([]string(nil), stack...), key)

	if strings.EqualFold(commandKind(lookup.Command.Kind), "imm") {
		if e.Runner == nil {
			return "", true, nextStack, fmt.Errorf("IMM runner is not configured")
		}
		expanded, err := e.expandRequestAtDepth(ctx, execCtx, lookup.Command.Response, lookup.RawArgs, lookup.Args, depth+1, nextStack)
		if err != nil {
			return "", true, nextStack, err
		}
		result, err := e.Runner.Run(ctx, imm.Request{
			Source:    expanded.Source,
			Args:      expanded.Args,
			RawArgs:   expanded.RawArgs,
			UserID:    execCtx.UserID,
			ChannelID: execCtx.ChannelID,
			GuildID:   execCtx.GuildID,
		})
		if err != nil {
			return "", true, nextStack, err
		}
		if result.ExitCode != 0 || result.TimedOut || result.OutputTruncated {
			return "", true, nextStack, fmt.Errorf("%s", FormatImmFailure("IMMコマンド展開に失敗しました", result))
		}
		return strings.TrimRight(result.Stdout, "\r\n"), true, nextStack, nil
	}

	return strings.TrimRight(lookup.Command.Response, "\r\n"), true, nextStack, nil
}

func (e CommandExpander) lookupReference(guildID string, ref commandReference) (commandLookupResult, error) {
	if e.Client == nil || strings.TrimSpace(guildID) == "" {
		return commandLookupResult{}, nil
	}

	cmd, err := e.Client.GetCommand(guildID, ref.Text)
	if err != nil {
		return commandLookupResult{}, err
	}
	if commandMatchesReference(cmd, ref.Kind) {
		return commandLookupResult{Name: ref.Text, Command: cmd, Found: true}, nil
	}

	name, rawArgs := splitCommandReferenceText(ref.Text)
	if name == "" || name == ref.Text {
		return commandLookupResult{}, nil
	}
	cmd, err = e.Client.GetCommand(guildID, name)
	if err != nil || cmd == nil {
		return commandLookupResult{}, err
	}
	if !commandMatchesReference(cmd, ref.Kind) {
		return commandLookupResult{}, nil
	}
	args, err := SplitExpansionArgs(rawArgs)
	if err != nil {
		return commandLookupResult{}, err
	}
	return commandLookupResult{Name: name, Command: cmd, RawArgs: rawArgs, Args: args, Found: true}, nil
}

func (e CommandExpander) maxDepth() int {
	if e.MaxDepth <= 0 {
		return DefaultCommandExpansionLimit
	}
	return e.MaxDepth
}

func CommandExpansionTimeout(runner *imm.Runner) time.Duration {
	timeout := imm.DefaultTimeout
	if runner != nil && runner.Timeout > 0 {
		timeout = runner.Timeout
	}
	return time.Duration(DefaultCommandExpansionLimit+1) * (timeout + time.Second)
}

func parseCommandReference(value string) (commandReference, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 2 {
		return commandReference{}, false
	}
	switch value[0] {
	case '!':
		return commandReference{Prefix: "!", Kind: "text", Text: strings.TrimSpace(value[1:])}, strings.TrimSpace(value[1:]) != ""
	case '?':
		return commandReference{Prefix: "?", Kind: "imm", Text: strings.TrimSpace(value[1:])}, strings.TrimSpace(value[1:]) != ""
	default:
		return commandReference{}, false
	}
}

func parseExpandableCommandReference(value string) (commandReference, bool, bool) {
	ref, ok := parseCommandReference(value)
	if ok {
		return ref, false, true
	}
	ref, ok = parseParenthesizedCommandReference(value)
	if ok {
		return ref, true, true
	}
	return commandReference{}, false, false
}

func parseParenthesizedCommandReference(value string) (commandReference, bool) {
	value = strings.TrimSpace(value)
	if len(value) < 4 || value[0] != '(' || value[len(value)-1] != ')' {
		return commandReference{}, false
	}
	if !isSingleParenthesizedExpression(value) {
		return commandReference{}, false
	}
	return parseCommandReference(strings.TrimSpace(value[1 : len(value)-1]))
}

func isSingleParenthesizedExpression(value string) bool {
	depth := 0
	var quote rune
	escaped := false

	for i, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			continue
		}
		switch r {
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return false
			}
			if depth == 0 && i != len(value)-1 {
				return false
			}
		}
	}
	return depth == 0 && quote == 0
}

func commandMatchesReference(cmd *client.BotCommandResponse, kind string) bool {
	if cmd == nil {
		return false
	}
	return strings.EqualFold(commandKind(cmd.Kind), kind)
}

func commandKind(kind string) string {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return "text"
	}
	return kind
}

func (e CommandExpander) commandBody(execCtx CommandExecutionContext, refText string) (string, error) {
	ref, ok := parseCommandReference(refText)
	if !ok {
		return "", fmt.Errorf("command reference %q is invalid", strings.TrimSpace(refText))
	}
	lookup, err := e.lookupReference(execCtx.GuildID, ref)
	if err != nil {
		return "", err
	}
	if !lookup.Found {
		return "", fmt.Errorf("command reference %q was not found", strings.TrimSpace(refText))
	}
	return lookup.Command.Response, nil
}

func (e CommandExpander) commandInfoLiteral(execCtx CommandExecutionContext, refText string) (string, error) {
	ref, ok := parseCommandReference(refText)
	if !ok {
		return "", fmt.Errorf("command reference %q is invalid", strings.TrimSpace(refText))
	}
	lookup, err := e.lookupReference(execCtx.GuildID, ref)
	if err != nil {
		return "", err
	}
	if !lookup.Found {
		return "", fmt.Errorf("command reference %q was not found", strings.TrimSpace(refText))
	}
	return commandRecordLiteral(client.CommandRecord{
		Name:     lookup.Name,
		Kind:     commandKind(lookup.Command.Kind),
		Response: lookup.Command.Response,
	}), nil
}

func (e CommandExpander) commandsLiteral(execCtx CommandExecutionContext, filter string) (string, error) {
	filter, err := normalizeCommandFilter(filter)
	if err != nil {
		return "", err
	}
	if e.Client == nil || strings.TrimSpace(execCtx.GuildID) == "" {
		return "[]", nil
	}
	commands, err := e.Client.ListCommands(execCtx.GuildID)
	if err != nil {
		return "", err
	}

	parts := make([]string, 0, len(commands))
	for _, cmd := range commands {
		kind := commandKind(cmd.Kind)
		if filter != "all" && !strings.EqualFold(kind, filter) {
			continue
		}
		cmd.Kind = kind
		parts = append(parts, commandRecordLiteral(cmd))
	}
	return "[" + strings.Join(parts, ", ") + "]", nil
}

func normalizeCommandFilter(filter string) (string, error) {
	filter = strings.ToLower(strings.TrimSpace(filter))
	switch filter {
	case "", "all", "any":
		return "all", nil
	case "text", "imm":
		return filter, nil
	default:
		return "", fmt.Errorf("bot_commands filter must be \"all\", \"text\", or \"imm\"")
	}
}

func commandRecordLiteral(cmd client.CommandRecord) string {
	kind := commandKind(cmd.Kind)
	prefix := "!"
	if strings.EqualFold(kind, "imm") {
		prefix = "?"
	}
	return "{" +
		`"name": ` + strconv.Quote(cmd.Name) + ", " +
		`"prefix": ` + strconv.Quote(prefix) + ", " +
		`"kind": ` + strconv.Quote(kind) + ", " +
		`"body": ` + strconv.Quote(cmd.Response) + ", " +
		`"tags": ` + stringArrayLiteral(cmd.Tags) +
		"}"
}

func stringArrayLiteral(values []string) string {
	if len(values) == 0 {
		return "[]"
	}
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, strconv.Quote(value))
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func splitCommandReferenceText(commandText string) (string, string) {
	commandText = strings.TrimSpace(commandText)
	idx := strings.IndexFunc(commandText, unicode.IsSpace)
	if idx < 0 {
		return commandText, ""
	}
	return commandText[:idx], strings.TrimSpace(commandText[idx:])
}

func joinRawArgs(args []string) string {
	parts := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" || strings.ContainsAny(arg, " \t\r\n\"'\\") {
			parts = append(parts, strconv.Quote(arg))
		} else {
			parts = append(parts, arg)
		}
	}
	return strings.Join(parts, " ")
}

func expansionArgs(rawArgs string, args []string) ([]expansionArg, error) {
	if strings.TrimSpace(rawArgs) != "" {
		return parseExpansionArgs(rawArgs)
	}
	tokens := make([]expansionArg, 0, len(args))
	for _, arg := range args {
		tokens = append(tokens, expansionArg{Value: arg, Expandable: true})
	}
	return tokens, nil
}

func SplitExpansionArgs(input string) ([]string, error) {
	tokens, err := parseExpansionArgs(input)
	if err != nil {
		return nil, err
	}
	args := make([]string, 0, len(tokens))
	for _, token := range tokens {
		args = append(args, token.Value)
	}
	return args, nil
}

func parseExpansionArgs(input string) ([]expansionArg, error) {
	var args []expansionArg
	var current strings.Builder
	var quote rune
	escaped := false
	groupEscaped := false
	inToken := false
	tokenExpandable := false
	groupDepth := 0

	flush := func() {
		if !inToken {
			return
		}
		args = append(args, expansionArg{Value: current.String(), Expandable: tokenExpandable})
		current.Reset()
		inToken = false
		tokenExpandable = false
	}

	for i, r := range input {
		if groupDepth > 0 {
			current.WriteRune(r)
			inToken = true
			tokenExpandable = true
			if groupEscaped {
				groupEscaped = false
				continue
			}
			if r == '\\' {
				groupEscaped = true
				continue
			}
			if quote != 0 {
				if r == quote {
					quote = 0
				}
				continue
			}
			if r == '"' || r == '\'' {
				quote = r
				continue
			}
			if r == '(' {
				groupDepth++
			}
			if r == ')' {
				groupDepth--
			}
			continue
		}
		if quote != 0 {
			if escaped {
				current.WriteRune(r)
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == quote {
				quote = 0
				continue
			}
			current.WriteRune(r)
			continue
		}
		if escaped {
			current.WriteRune(r)
			escaped = false
			inToken = true
			tokenExpandable = true
			continue
		}
		if r == '\\' {
			escaped = true
			inToken = true
			tokenExpandable = true
			continue
		}
		if r == '"' || r == '\'' {
			quote = r
			inToken = true
			continue
		}
		if unicode.IsSpace(r) {
			flush()
			continue
		}
		if r == '(' && !inToken && beginsCommandExpressionAt(input, i) {
			groupDepth = 1
			current.WriteRune(r)
			inToken = true
			tokenExpandable = true
			continue
		}
		current.WriteRune(r)
		inToken = true
		tokenExpandable = true
	}

	if escaped {
		current.WriteRune('\\')
	}
	if quote != 0 {
		return nil, fmt.Errorf("unterminated quote")
	}
	if groupDepth != 0 {
		return nil, fmt.Errorf("unterminated command expression")
	}
	flush()
	return args, nil
}

func beginsCommandExpressionAt(input string, index int) bool {
	if index >= len(input) || input[index] != '(' {
		return false
	}
	i := skipSpaces(input, index+1)
	return i < len(input) && (input[i] == '!' || input[i] == '?')
}

func hasFunctionNameAt(source string, index int, name string) bool {
	if !strings.HasPrefix(source[index:], name) {
		return false
	}
	if index > 0 && isIdentifierRune(rune(source[index-1])) {
		return false
	}
	next := index + len(name)
	if next < len(source) && isIdentifierRune(rune(source[next])) {
		return false
	}
	return true
}

func parseBotStringCall(source string, index int, name string) (string, int, error) {
	i := index + len(name)
	i = skipSpaces(source, i)
	if i >= len(source) || source[i] != '(' {
		return "", index, fmt.Errorf("%s must be called as %s(\"!command\")", name, name)
	}
	i++
	i = skipSpaces(source, i)
	value, next, err := parseStringLiteral(source, i)
	if err != nil {
		return "", index, err
	}
	next = skipSpaces(source, next)
	if next >= len(source) || source[next] != ')' {
		return "", index, fmt.Errorf("%s call must close with )", name)
	}
	return value, next + 1, nil
}

func parseBotOptionalStringCall(source string, index int, name, defaultValue string) (string, int, error) {
	i := index + len(name)
	i = skipSpaces(source, i)
	if i >= len(source) || source[i] != '(' {
		return "", index, fmt.Errorf("%s must be called as %s(\"all\")", name, name)
	}
	i++
	i = skipSpaces(source, i)
	if i < len(source) && source[i] == ')' {
		return defaultValue, i + 1, nil
	}
	value, next, err := parseStringLiteral(source, i)
	if err != nil {
		return "", index, err
	}
	next = skipSpaces(source, next)
	if next >= len(source) || source[next] != ')' {
		return "", index, fmt.Errorf("%s call must close with )", name)
	}
	return value, next + 1, nil
}

func parseStringLiteral(source string, index int) (string, int, error) {
	if index >= len(source) || (source[index] != '"' && source[index] != '\'') {
		return "", index, fmt.Errorf("bot API call expects a string literal")
	}
	quote := source[index]
	var value strings.Builder
	for i := index + 1; i < len(source); i++ {
		if source[i] == '\\' {
			if i+1 >= len(source) {
				value.WriteByte('\\')
				continue
			}
			i++
			value.WriteByte(source[i])
			continue
		}
		if source[i] == quote {
			return value.String(), i + 1, nil
		}
		value.WriteByte(source[i])
	}
	return "", index, fmt.Errorf("unterminated string literal in bot_command")
}

func skipSpaces(source string, index int) int {
	for index < len(source) {
		r := rune(source[index])
		if !unicode.IsSpace(r) {
			break
		}
		index++
	}
	return index
}

func copyQuoted(out *strings.Builder, source string, index int) int {
	quote := source[index]
	out.WriteByte(source[index])
	for i := index + 1; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] == '\\' && i+1 < len(source) {
			i++
			out.WriteByte(source[i])
			continue
		}
		if source[i] == quote {
			return i + 1
		}
	}
	return len(source)
}

func copyLineComment(out *strings.Builder, source string, index int) int {
	for i := index; i < len(source); i++ {
		out.WriteByte(source[i])
		if source[i] == '\n' {
			return i + 1
		}
	}
	return len(source)
}

func copyBlockComment(out *strings.Builder, source string, index int) int {
	for i := index; i < len(source); i++ {
		out.WriteByte(source[i])
		if i > index && source[i-1] == '*' && source[i] == '/' {
			return i + 1
		}
	}
	return len(source)
}

func isIdentifierRune(r rune) bool {
	return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r)
}
