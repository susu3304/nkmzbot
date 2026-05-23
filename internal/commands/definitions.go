package commands

import "github.com/bwmarrin/discordgo"

func GetCommands() []*discordgo.ApplicationCommand {
	sourceMaxLength := 4000
	return []*discordgo.ApplicationCommand{
		{
			Name:         "imm",
			Description:  "IMMコードの実行とIMMカスタムコマンドを管理します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "run",
					Description: "IMMコードをその場で実行します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "source",
							Description: "実行するIMMコード（コードブロック可）",
							Required:    true,
							MaxLength:   sourceMaxLength,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "args",
							Description: "bot_argsに渡す引数（空白区切り、引用符可）",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "check",
					Description: "IMMコードを構文チェックします",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "source",
							Description: "チェックするIMMコード（コードブロック可）",
							Required:    true,
							MaxLength:   sourceMaxLength,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Name:        "command",
					Description: "IMMカスタムコマンドを管理します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Name:        "add",
							Description: "IMMカスタムコマンドを追加します",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "name",
									Description: "コマンド名（?は不要）",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "source",
									Description: "IMMソース（bot_argsを使用できます）",
									Required:    true,
									MaxLength:   sourceMaxLength,
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "tags",
									Description: "タグ（カンマ区切り、省略可）",
									Required:    false,
								},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Name:        "update",
							Description: "IMMカスタムコマンドを更新します",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "name",
									Description: "更新するコマンド名",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "source",
									Description: "新しいIMMソース",
									Required:    true,
									MaxLength:   sourceMaxLength,
								},
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "tags",
									Description: "タグ（カンマ区切り、省略可）",
									Required:    false,
								},
							},
						},
						{
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Name:        "remove",
							Description: "IMMカスタムコマンドを削除します",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "name",
									Description: "削除するコマンド名",
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
		{
			Name:         "add",
			Description:  "新しいコマンドを追加します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "コマンド名",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "response",
					Description: "返答内容",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tags",
					Description: "タグ（カンマ区切り、省略可）",
					Required:    false,
				},
			},
		},
		{
			Name:         "remove",
			Description:  "コマンドを削除します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "削除するコマンド名",
					Required:    true,
				},
			},
		},
		{
			Name:         "update",
			Description:  "コマンドを更新します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "更新するコマンド名",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "response",
					Description: "新しい返答内容",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tags",
					Description: "タグ（カンマ区切り、省略可）",
					Required:    false,
				},
			},
		},
		{
			Name:         "addbulk",
			Description:  "複数のコマンドを一括で追加します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "commands",
					Description: "コマンドリスト（形式: !cmd1: content1 改行 !cmd2: content2 ...）",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tags",
					Description: "全コマンドに付けるタグ（カンマ区切り、省略可）",
					Required:    false,
				},
			},
		},
		{
			Name:         "jikan",
			Description:  "スケジュール実行を管理します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "add",
					Description: "指定された時間にコマンドを実行します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "command",
							Description: "実行するコマンド（例: nomikai start, または任意のメッセージ）",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "time",
							Description: "実行する時間（HH:MM または YYYY-MM-DD HH:MM）",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "repeat",
							Description: "毎日繰り返すかどうか",
							Required:    false,
						},
					},
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "list",
					Description: "予約されているコマンド一覧を表示します",
				},
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "delete",
					Description: "予約されているコマンドを削除します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "id",
							Description: "削除するタスクのID",
							Required:    true,
						},
					},
				},
			},
		},
		{
			Name:         "list",
			Description:  "登録されているコマンド一覧を表示します",
			DMPermission: boolPtr(false),
		},
		{
			Name:         "ramdom",
			Description:  "登録されているコマンドからランダムに1つ返します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "tag",
					Description: "このタグのコマンドから選びます（省略可）",
					Required:    false,
				},
			},
		},
		{
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
		},
		{
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
		},
		{
			Name:         "wallet",
			Description:  "個人間の送金・請求を記録します",
			DMPermission: boolPtr(false),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Name:        "pay",
					Description: "自分が支払う送金を記録します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "受け取り相手",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "金額（円）",
							Required:    true,
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
					Name:        "req",
					Description: "相手への請求を記録します",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "user",
							Description: "支払ってほしい相手",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "金額（円）",
							Required:    true,
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
		},
		{
			Name: "Register as Response",
			Type: discordgo.MessageApplicationCommand,
		},
		{
			Name: "Register as IMM",
			Type: discordgo.MessageApplicationCommand,
		},
		{
			Name: "Run as IMM",
			Type: discordgo.MessageApplicationCommand,
		},
	}
}

func boolPtr(b bool) *bool {
	return &b
}
