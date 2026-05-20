package commands

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/susu3304/nkmzbot/internal/client"
	"github.com/susu3304/nkmzbot/internal/db"
	"github.com/susu3304/nkmzbot/internal/nomikai"
)

// In-memory storage for active scheduled task timers
type activeTask struct {
	ID        int
	Command   string
	Time      time.Time
	Repeat    bool
	ChannelID string
	GuildID   int64
	UserID    string
	timer     *time.Timer
}

var (
	activeTasks = make(map[int]*activeTask)
	tasksMu     sync.Mutex
	// JST timezone (UTC+9)
	jst = time.FixedZone("JST", 9*60*60)
)

const scheduledTaskPollInterval = 30 * time.Second

// RestoreScheduledTasks loads all scheduled tasks from the database and schedules them
func RestoreScheduledTasks(ctx context.Context, s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client) error {
	if err := syncScheduledTasks(ctx, s, svc, database, cli, false); err != nil {
		return err
	}

	tasksMu.Lock()
	count := len(activeTasks)
	tasksMu.Unlock()
	log.Printf("Restored %d scheduled tasks", count)
	return nil
}

func StartScheduledTaskPolling(s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, interval time.Duration) context.CancelFunc {
	if interval <= 0 {
		interval = scheduledTaskPollInterval
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				pollCtx, pollCancel := context.WithTimeout(ctx, 10*time.Second)
				if err := SyncScheduledTasks(pollCtx, s, svc, database, cli); err != nil {
					log.Printf("Failed to poll scheduled tasks: %v", err)
				}
				pollCancel()
			}
		}
	}()

	return cancel
}

func SyncScheduledTasks(ctx context.Context, s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client) error {
	return syncScheduledTasks(ctx, s, svc, database, cli, true)
}

func syncScheduledTasks(ctx context.Context, s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, runExpiredNonRepeating bool) error {
	dbTasks, err := database.ListAllScheduledTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load scheduled tasks: %w", err)
	}

	now := time.Now()
	desired := make(map[int]*activeTask, len(dbTasks))

	for _, dbTask := range dbTasks {
		task, ok := prepareScheduledTask(ctx, database, dbTask, now, runExpiredNonRepeating)
		if ok {
			desired[task.ID] = task
		}
	}

	reconcileActiveTasks(s, svc, database, cli, desired)
	return nil
}

func prepareScheduledTask(ctx context.Context, database *db.DB, dbTask *db.ScheduledTask, now time.Time, runExpiredNonRepeating bool) (*activeTask, bool) {
	if dbTask.Time.Before(now) && !dbTask.Repeat {
		if runExpiredNonRepeating {
			return newActiveTask(dbTask), true
		}
		if err := database.DeleteScheduledTask(ctx, dbTask.ID); err != nil {
			log.Printf("Failed to delete expired task %d: %v", dbTask.ID, err)
		}
		return nil, false
	}

	if dbTask.Time.Before(now) && dbTask.Repeat {
		nextTime := nextDailyOccurrence(dbTask.Time, now)
		dbTask.Time = nextTime
		if err := database.UpdateScheduledTaskTime(ctx, dbTask.ID, nextTime); err != nil {
			log.Printf("Failed to update task %d time: %v", dbTask.ID, err)
			return nil, false
		}
	}

	return newActiveTask(dbTask), true
}

func newActiveTask(dbTask *db.ScheduledTask) *activeTask {
	return &activeTask{
		ID:        dbTask.ID,
		Command:   dbTask.Command,
		Time:      dbTask.Time,
		Repeat:    dbTask.Repeat,
		ChannelID: dbTask.ChannelID,
		GuildID:   dbTask.GuildID,
		UserID:    dbTask.UserID,
	}
}

func nextDailyOccurrence(scheduled, now time.Time) time.Time {
	if scheduled.After(now) {
		return scheduled
	}
	daysToAdd := int(now.Sub(scheduled).Hours()/24) + 1
	return scheduled.Add(time.Duration(daysToAdd) * 24 * time.Hour)
}

func reconcileActiveTasks(s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, desired map[int]*activeTask) {
	for _, task := range desired {
		activateScheduledTask(s, svc, database, cli, task)
	}

	tasksMu.Lock()
	defer tasksMu.Unlock()
	for id, task := range activeTasks {
		if _, ok := desired[id]; ok {
			continue
		}
		if task.timer != nil {
			task.timer.Stop()
		}
		delete(activeTasks, id)
	}
}

func HandleJikan(s *discordgo.Session, i *discordgo.InteractionCreate, svc *nomikai.Service, database *db.DB, cli *client.Client) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		respondText(s, i, "サブコマンドを指定してください")
		return
	}

	subCmd := options[0]

	switch subCmd.Name {
	case "add":
		handleJikanAdd(s, i, subCmd.Options, svc, database, cli)
	case "list":
		handleJikanList(s, i)
	case "delete":
		handleJikanDelete(s, i, subCmd.Options, database)
	}
}

func handleJikanAdd(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, svc *nomikai.Service, database *db.DB, cli *client.Client) {
	cmdStr := getStringOption(options, "command")
	timeStr := getStringOption(options, "time")
	repeatOpt := getBoolOption(options, "repeat")

	if cmdStr == nil || timeStr == nil {
		respondText(s, i, "コマンドと時間を指定してください")
		return
	}

	isRepeat := false
	if repeatOpt != nil {
		isRepeat = *repeatOpt
	}

	targetTime, err := parseTime(*timeStr)
	if err != nil {
		respondText(s, i, fmt.Sprintf("時間の形式が正しくありません: %v (例: 18:00, 2025-12-26 18:00)", err))
		return
	}

	now := time.Now()
	if targetTime.Before(now) {
		respondText(s, i, "指定された時間は既に過ぎています")
		return
	}

	channelID := i.ChannelID
	guildID := i.GuildID
	userID := i.Member.User.ID

	// Parse guildID to int64
	gid, err := strconv.ParseInt(guildID, 10, 64)
	if err != nil {
		respondText(s, i, "ギルドIDの解析に失敗しました")
		return
	}

	// Save to database
	ctx := context.Background()
	dbTask, err := database.AddScheduledTask(ctx, *cmdStr, targetTime, isRepeat, channelID, gid, userID)
	if err != nil {
		respondText(s, i, fmt.Sprintf("タスクの保存に失敗しました: %v", err))
		return
	}

	activateScheduledTask(s, svc, database, cli, newActiveTask(dbTask))

	// Display time in JST for user
	jstTime := targetTime.In(jst)
	msg := fmt.Sprintf("ID: %d\nコマンド `%s` を %s に実行するように予約しました", dbTask.ID, *cmdStr, jstTime.Format("2006-01-02 15:04"))
	if isRepeat {
		msg += "（毎日繰り返し）"
	}
	respondText(s, i, msg)
}

func handleJikanList(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Parse guildID to int64
	gid, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		respondText(s, i, "ギルドIDの解析に失敗しました")
		return
	}

	tasksMu.Lock()
	defer tasksMu.Unlock()

	if len(activeTasks) == 0 {
		respondText(s, i, "予約されているコマンドはありません")
		return
	}

	var b strings.Builder
	b.WriteString("予約コマンド一覧:\n")

	for _, t := range activeTasks {
		if t.GuildID != gid {
			continue
		}

		repeatStr := ""
		if t.Repeat {
			repeatStr = " (毎日)"
		}
		// Display time in JST
		jstTime := t.Time.In(jst)
		fmt.Fprintf(&b, "- ID: %d | %s | `%s`%s\n", t.ID, jstTime.Format("2006-01-02 15:04"), t.Command, repeatStr)
	}

	if b.Len() == len("予約コマンド一覧:\n") {
		respondText(s, i, "このサーバーで予約されているコマンドはありません")
		return
	}

	respondText(s, i, b.String())
}

func handleJikanDelete(s *discordgo.Session, i *discordgo.InteractionCreate, options []*discordgo.ApplicationCommandInteractionDataOption, database *db.DB) {
	taskIDOpt := getIntegerOption(options, "id")

	if taskIDOpt == nil {
		respondText(s, i, "タスクIDを指定してください")
		return
	}

	taskID := int(*taskIDOpt)

	// Parse guildID to int64
	gid, err := strconv.ParseInt(i.GuildID, 10, 64)
	if err != nil {
		respondText(s, i, "ギルドIDの解析に失敗しました")
		return
	}

	// Check if task exists and belongs to this guild
	tasksMu.Lock()
	task, exists := activeTasks[taskID]
	tasksMu.Unlock()

	if !exists {
		respondText(s, i, fmt.Sprintf("ID %d のタスクが見つかりません", taskID))
		return
	}

	if task.GuildID != gid {
		respondText(s, i, "このサーバーのタスクではありません")
		return
	}

	// Delete from database (without holding the mutex)
	ctx := context.Background()
	if err := database.DeleteScheduledTask(ctx, taskID); err != nil {
		respondText(s, i, fmt.Sprintf("タスクの削除に失敗しました: %v", err))
		return
	}

	removeActiveTask(taskID)

	respondText(s, i, fmt.Sprintf("タスク ID %d を削除しました", taskID))
}

func getIntegerOption(opts []*discordgo.ApplicationCommandInteractionDataOption, name string) *int64 {
	for _, o := range opts {
		if o.Name == name {
			v := o.IntValue()
			return &v
		}
	}
	return nil
}

func activateScheduledTask(s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, task *activeTask) {
	tasksMu.Lock()
	defer tasksMu.Unlock()

	if existing, ok := activeTasks[task.ID]; ok {
		if sameActiveTask(existing, task) {
			return
		}
		if existing.timer != nil {
			existing.timer.Stop()
		}
	}

	now := time.Now()
	duration := task.Time.Sub(now)
	if duration < 0 {
		duration = 0
	}

	activeTasks[task.ID] = task
	task.timer = time.AfterFunc(duration, func() {
		executeActiveTask(s, svc, database, cli, task.ID)
	})
}

func sameActiveTask(a, b *activeTask) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.ID == b.ID &&
		a.Command == b.Command &&
		a.Time.Equal(b.Time) &&
		a.Repeat == b.Repeat &&
		a.ChannelID == b.ChannelID &&
		a.GuildID == b.GuildID &&
		a.UserID == b.UserID
}

func executeActiveTask(s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, taskID int) {
	tasksMu.Lock()
	task, exists := activeTasks[taskID]
	tasksMu.Unlock()
	if !exists {
		return
	}

	guildIDStr := strconv.FormatInt(task.GuildID, 10)
	executeScheduledCommand(s, svc, database, cli, task.ChannelID, guildIDStr, task.UserID, task.Command)

	if !isCurrentActiveTask(taskID, task) {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if task.Repeat {
		nextTime := nextDailyOccurrence(task.Time, time.Now())
		if err := database.UpdateScheduledTaskTime(ctx, task.ID, nextTime); err != nil {
			log.Printf("Failed to update scheduled task time: %v", err)
			return
		}

		if !isCurrentActiveTask(taskID, task) {
			return
		}
		nextTask := *task
		nextTask.Time = nextTime
		nextTask.timer = nil
		activateScheduledTask(s, svc, database, cli, &nextTask)
		return
	}

	if err := database.DeleteScheduledTask(ctx, task.ID); err != nil {
		log.Printf("Failed to delete scheduled task: %v", err)
	}
	removeActiveTaskIfCurrent(taskID, task)
}

func isCurrentActiveTask(taskID int, task *activeTask) bool {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	return activeTasks[taskID] == task
}

func removeActiveTask(taskID int) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	if task, ok := activeTasks[taskID]; ok {
		if task.timer != nil {
			task.timer.Stop()
		}
		delete(activeTasks, taskID)
	}
}

func removeActiveTaskIfCurrent(taskID int, expected *activeTask) {
	tasksMu.Lock()
	defer tasksMu.Unlock()
	task, ok := activeTasks[taskID]
	if !ok || task != expected {
		return
	}
	if task.timer != nil {
		task.timer.Stop()
	}
	delete(activeTasks, taskID)
}

func parseTime(input string) (time.Time, error) {
	now := time.Now().In(jst)

	// Try HH:MM format
	if t, err := time.ParseInLocation("15:04", input, jst); err == nil {
		target := time.Date(now.Year(), now.Month(), now.Day(), t.Hour(), t.Minute(), 0, 0, jst)
		if target.Before(now) {
			target = target.Add(24 * time.Hour)
		}
		// Convert to UTC before returning
		return target.UTC(), nil
	}

	// Try YYYY-MM-DD HH:MM format
	if t, err := time.ParseInLocation("2006-01-02 15:04", input, jst); err == nil {
		// Convert to UTC before returning
		return t.UTC(), nil
	}

	return time.Time{}, fmt.Errorf("unsupported format")
}

func executeScheduledCommand(s *discordgo.Session, svc *nomikai.Service, database *db.DB, cli *client.Client, channelID, guildIDStr, userID, cmdStr string) {
	parts := strings.Fields(cmdStr)
	if len(parts) == 0 {
		return
	}

	mainCmd := parts[0]

	// Check for custom command (starts with !)
	if strings.HasPrefix(mainCmd, "!") && len(mainCmd) > 1 {
		cmdName := mainCmd[1:]
		// Use API client instead of DB
		resp, err := cli.GetCommandResponse(guildIDStr, cmdName)
		if err == nil && resp != "" {
			if _, err := s.ChannelMessageSend(channelID, resp); err == nil {
				go func() {
					_ = cli.RecordCommandUsage(guildIDStr, cmdName, client.CommandUsageInput{
						ActorID:   userID,
						ChannelID: channelID,
						Source:    "bot-scheduled",
					})
				}()
			}
			return
		}
	}

	switch mainCmd {
	case "nomikai":
		if len(parts) < 2 {
			s.ChannelMessageSend(channelID, "nomikai コマンドにはサブコマンドが必要です")
			return
		}
		subCmd := parts[1]
		ctx := context.Background()

		switch subCmd {
		case "start":
			gid, _ := strconv.ParseInt(guildIDStr, 10, 64)
			err := svc.StartSession(ctx, channelID, gid, userID, 1, "organizer")
			if err != nil {
				s.ChannelMessageSend(channelID, fmt.Sprintf("予約実行エラー (nomikai start): %v", err))
			} else {
				s.ChannelMessageSend(channelID, "予約実行: 飲み会セッションを開始しました")
			}
		case "stop":
			err := svc.StopSession(ctx, channelID)
			if err != nil {
				s.ChannelMessageSend(channelID, fmt.Sprintf("予約実行エラー (nomikai stop): %v", err))
			} else {
				s.ChannelMessageSend(channelID, "予約実行: 飲み会セッションを終了しました")
			}
		default:
			s.ChannelMessageSend(channelID, fmt.Sprintf("予約実行: 未対応の nomikai サブコマンドです: %s", subCmd))
		}
	default:
		// For other commands, just send the message
		s.ChannelMessageSend(channelID, cmdStr)
	}
}
