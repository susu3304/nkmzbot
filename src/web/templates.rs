use askama::Template;

use crate::web::oauth::DiscordGuild;

#[derive(Template)]
#[template(source = r#"
<!doctype html>
<html><head><meta charset='utf-8'><title>nkmzbot</title></head>
<body>
  <h1>nkmzbot Web</h1>
  <p><a href='/login'>Discordでログイン</a></p>
</body></html>
"#, ext = "html" )]
pub struct HomeTemplate {}

#[derive(Template)]
#[template(source = r#"
<!doctype html>
<html><head><meta charset='utf-8'><title>Dashboard</title></head>
<body>
  <header>
    <nav>
      <a href='/dashboard'>Dashboard</a> | <a href='/logout'>Logout</a>
    </nav>
  </header>
  <h2>Guild一覧 {{ username.as_deref().unwrap_or("") }}</h2>
  <ul>
  {% for g in guilds %}
    <li><a href="/guilds/{{ g.id }}/commands">{{ g.name }}</a></li>
  {% endfor %}
  </ul>
</body></html>
"#, ext = "html" )]
pub struct DashboardTemplate {
    pub username: Option<String>,
    pub guilds: Vec<DiscordGuild>,
}

#[derive(askama::Template)]
#[template(source = r#"
<!doctype html>
<html><head><meta charset='utf-8'><title>Commands</title>
  <style>
    table { border-collapse: collapse; width: 100%; }
    th, td { border: 1px solid #ddd; padding: 8px; }
    th { background: #f7f7f7; }
    textarea { width: 100%; height: 4em; }
  </style>
</head>
<body>
  <header>
    <nav>
      <a href='/dashboard'>← Back</a>
    </nav>
  </header>
  <h2>Guild {{ guild_id }} のコマンド</h2>

  <form method='get'>
    <input type='text' name='q' placeholder='検索' value='{{ q }}'>
    <button type='submit'>検索</button>
  </form>

  <h3>追加</h3>
  <form method='post' action='/guilds/{{ guild_id }}/commands/add'>
    <input type='hidden' name='csrf' value='{{ csrf }}'>
    <div>
      <label>name</label>
      <input name='name' required>
    </div>
    <div>
      <label>response</label>
      <textarea name='response' required></textarea>
    </div>
    <button type='submit'>追加</button>
  </form>

  <h3>一覧</h3>
  <table>
    <thead>
      <tr><th>選択</th><th>name</th><th>response</th><th>更新</th></tr>
    </thead>
    <tbody>
    {% for c in commands %}
      <tr>
        <td><input type='checkbox' name='names' value='{{ c.name }}' form='bulk-form'></td>
        <td>{{ c.name }}</td>
        <td>
          <form method='post' action='/guilds/{{ guild_id }}/commands/update'>
            <input type='hidden' name='csrf' value='{{ csrf }}'>
            <input type='hidden' name='name' value='{{ c.name }}'>
            <textarea name='response'>{{ c.response }}</textarea>
            <button type='submit'>更新</button>
          </form>
        </td>
        <td></td>
      </tr>
    {% endfor %}
    </tbody>
  </table>

  <form id='bulk-form' method='post' action='/guilds/{{ guild_id }}/commands/bulk-delete'>
    <input type='hidden' name='csrf' value='{{ csrf }}'>
    <button type='submit'>選択を削除</button>
  </form>

</body></html>
"#, ext = "html" )]
pub struct CommandsTemplate {
    pub guild_id: i64,
    pub q: String,
    pub commands: Vec<CmdRow>,
    pub csrf: String,
}

#[derive(Clone)]
pub struct CmdRow { pub name: String, pub response: String }
