use askama::Template;

use crate::web::oauth::DiscordGuild;

#[derive(Template)]
#[template(source = r#"
<!doctype html>
<html lang='ja'>
  <head>
    <meta charset='utf-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1'>
    <title>nkmzbot</title>
    <link rel='preconnect' href='https://cdn.jsdelivr.net'>
    <link rel='stylesheet' href='https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css'>
    <style>
      :root { --brand: #6c68ff; }
      html { font-size: 15px; }
      @media (min-width: 1200px) { html { font-size: 16px; } }
      body { line-height: 1.45; }
      main.container { max-width: 960px; }
      header .brand { font-weight: 700; letter-spacing: .2px; }
      header.container { padding: .25rem 0; }
      nav { margin: .25rem 0; }
      .hero { padding: 4rem 0 2rem; text-align: center; }
      .hero h1 { font-size: clamp(1.6rem, 3vw + .8rem, 2.4rem); margin-bottom: .25rem; }
      .hero p { color: var(--muted-color); }
      button, [role='button'], input, select, textarea { font-size: .95rem; }
    </style>
  </head>
  <body>
    <header class='container'>
      <nav>
        <ul>
          <li class='brand'><strong>nkmzbot</strong></li>
        </ul>
        <ul>
          <li><a href='/login' role='button' class='contrast'>Discordでログイン</a></li>
        </ul>
      </nav>
    </header>
    <main id='app' class='container'>
      <section class='hero'>
        <h1>nkmzbot Web</h1>
        <p>Discord ログインして、ギルドのカスタムコマンドをかんたん管理。</p>
        <p>
          <a href='/login' role='button' class='primary'>Discordでログイン</a>
        </p>
      </section>
    </main>
    <script>
      // Lightweight SPA-like navigation (intercept internal links/forms)
      (() => {
        const appSel = '#app';
        const isInternal = (url) => {
          try { const u = new URL(url, location.href); return u.origin === location.origin; } catch { return false; }
        };
        const swapContent = async (response, pushUrl) => {
          const html = await response.text();
          const doc = new DOMParser().parseFromString(html, 'text/html');
          const next = doc.querySelector(appSel);
          if (!next) return false;
          const current = document.querySelector(appSel);
          if (!current) return false;
          current.replaceWith(next);
          const t = doc.querySelector('title');
          if (t) document.title = t.textContent || document.title;
          if (pushUrl) history.pushState({}, '', pushUrl);
          // Re-run any per-page enhancements if needed in the future.
          return true;
        };
        const navTo = async (url, opts = {}) => {
          try {
            const res = await fetch(url, { credentials: 'include', redirect: 'follow', ...opts, headers: { 'X-Requested-With': 'fetch', ...(opts.headers||{}) } });
            if (!res.ok) { location.href = url; return; }
            const ok = await swapContent(res, opts.method && opts.method !== 'GET' ? res.url : url);
            if (!ok) location.href = url;
          } catch (_) { location.href = url; }
        };
        document.addEventListener('click', (e) => {
          const a = e.target.closest('a');
          if (!a) return;
          if (a.hasAttribute('download') || a.target && a.target !== '' && a.target !== '_self') return;
          const href = a.getAttribute('href');
          if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
          if (!isInternal(href) || a.dataset.noSpa === 'true') return;
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
          e.preventDefault();
          navTo(href);
        });
        document.addEventListener('submit', (e) => {
          const form = e.target;
          if (!(form instanceof HTMLFormElement)) return;
          // Only intercept forms inside app container
          if (!form.closest(appSel)) return;
          e.preventDefault();
          const method = (form.method || 'GET').toUpperCase();
          const action = form.action || location.href;
          const body = new FormData(form);
          navTo(action, { method, body });
        });
        window.addEventListener('popstate', () => navTo(location.href));
      })();
    </script>
  </body>
</html>
"#, ext = "html" )]
pub struct HomeTemplate {}

#[derive(Template)]
#[template(source = r#"
<!doctype html>
<html lang='ja'>
  <head>
    <meta charset='utf-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1'>
    <title>Dashboard - nkmzbot</title>
    <link rel='preconnect' href='https://cdn.jsdelivr.net'>
    <link rel='stylesheet' href='https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css'>
    <style>
      html { font-size: 15px; }
      @media (min-width: 1200px) { html { font-size: 16px; } }
      body { line-height: 1.45; }
      main.container { max-width: 1024px; }
      .grid { grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); }
      article > header { font-weight: 600; }
      h2 { font-size: 1.25rem; }
      header.container { padding: .25rem 0; }
      nav { margin: .25rem 0; }
      button, [role='button'], input, select, textarea { font-size: .95rem; }
    </style>
  </head>
  <body>
    <header class='container'>
      <nav>
        <ul>
          <li><a href='/' class='contrast'><strong>nkmzbot</strong></a></li>
          <li><a href='/dashboard'>Dashboard</a></li>
        </ul>
        <ul>
          <li>{{ username.as_deref().unwrap_or("") }}</li>
          <li><a href='/logout' role='button' class='secondary'>Logout</a></li>
        </ul>
      </nav>
    </header>
    <main id='app' class='container'>
      <h2>Guild 一覧</h2>
      {% if guilds.len() == 0 %}
        <article>
          <header>表示できるギルドがありません</header>
          <p>このアプリでコマンドが登録されているギルドのみが表示されます。</p>
        </article>
      {% else %}
        <ul role='list' class='grid'>
        {% for g in guilds %}
          <li>
            <article>
              <header>{{ g.name }}</header>
              <footer>
                <a href="/guilds/{{ g.id }}/commands" role='button' class='primary'>管理する</a>
              </footer>
            </article>
          </li>
        {% endfor %}
        </ul>
      {% endif %}
    </main>
    <script>
      // Shared SPA navigation (same as on Home)
      (() => {
        const appSel = '#app';
        const isInternal = (url) => { try { const u = new URL(url, location.href); return u.origin === location.origin; } catch { return false; } };
        const swapContent = async (response, pushUrl) => {
          const html = await response.text();
          const doc = new DOMParser().parseFromString(html, 'text/html');
          const next = doc.querySelector(appSel);
          if (!next) return false;
          const current = document.querySelector(appSel);
          if (!current) return false;
          current.replaceWith(next);
          const t = doc.querySelector('title');
          if (t) document.title = t.textContent || document.title;
          if (pushUrl) history.pushState({}, '', pushUrl);
          return true;
        };
        const navTo = async (url, opts = {}) => {
          try {
            const res = await fetch(url, { credentials: 'include', redirect: 'follow', ...opts, headers: { 'X-Requested-With': 'fetch', ...(opts.headers||{}) } });
            if (!res.ok) { location.href = url; return; }
            const ok = await swapContent(res, opts.method && opts.method !== 'GET' ? res.url : url);
            if (!ok) location.href = url;
          } catch (_) { location.href = url; }
        };
        document.addEventListener('click', (e) => {
          const a = e.target.closest('a');
          if (!a) return;
          if (a.hasAttribute('download') || a.target && a.target !== '' && a.target !== '_self') return;
          const href = a.getAttribute('href');
          if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
          if (!isInternal(href) || a.dataset.noSpa === 'true') return;
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
          e.preventDefault();
          navTo(href);
        });
        document.addEventListener('submit', (e) => {
          const form = e.target;
          if (!(form instanceof HTMLFormElement)) return;
          if (!form.closest(appSel)) return;
          e.preventDefault();
          const method = (form.method || 'GET').toUpperCase();
          const action = form.action || location.href;
          const body = new FormData(form);
          navTo(action, { method, body });
        });
        window.addEventListener('popstate', () => navTo(location.href));
      })();
    </script>
  </body>
</html>
"#, ext = "html" )]
pub struct DashboardTemplate {
    pub username: Option<String>,
    pub guilds: Vec<DiscordGuild>,
}

#[derive(askama::Template)]
#[template(source = r#"
<!doctype html>
<html lang='ja'>
  <head>
    <meta charset='utf-8'>
    <meta name='viewport' content='width=device-width, initial-scale=1'>
    <title>Commands - nkmzbot</title>
    <link rel='preconnect' href='https://cdn.jsdelivr.net'>
    <link rel='stylesheet' href='https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css'>
    <style>
      html { font-size: 15px; }
      @media (min-width: 1200px) { html { font-size: 16px; } }
      body { line-height: 1.45; }
      main.container { max-width: 1100px; }
      textarea { min-height: 4.5rem; }
      .toolbar { display: flex; gap: .5rem; align-items: center; }
      .toolbar input[type="text"] { flex: 1 1 auto; }
      .table-wrap { overflow-x: auto; }
      .muted { color: var(--muted-color); }
      table th, table td { padding: .4rem .5rem; }
      button, [role='button'], input, select, textarea { font-size: .95rem; }
      header.container { padding: .25rem 0; }
      nav { margin: .25rem 0; }
    </style>
  </head>
  <body>
    <header class='container'>
      <nav>
        <ul>
          <li><a href='/' class='contrast'><strong>nkmzbot</strong></a></li>
          <li><a href='/dashboard'>Dashboard</a></li>
        </ul>
        <ul>
          <li><a href='/dashboard'>&larr; Back</a></li>
        </ul>
      </nav>
    </header>
    <main id='app' class='container'>
      <h2>Guild {{ guild_id }} のコマンド</h2>

      <form method='get' class='toolbar'>
        <input type='text' name='q' placeholder='キーワードで検索' value='{{ q }}'>
        <button type='submit'>検索</button>
      </form>

      <article>
        <header>追加</header>
        <form method='post' action='/guilds/{{ guild_id }}/commands/add'>
          <input type='hidden' name='csrf' value='{{ csrf }}'>
          <div class='grid'>
            <label>
              name
              <input name='name' required placeholder='例: hello'>
            </label>
            <label>
              response
              <textarea name='response' required placeholder='例: Hello, world!'></textarea>
            </label>
          </div>
          <button type='submit' class='primary'>追加</button>
        </form>
      </article>

      <h3>一覧</h3>
      <div class='table-wrap'>
        <table>
          <thead>
            <tr><th style='width:4rem'><input type='checkbox' id='select-all'></th><th>name</th><th>response</th><th style='width:8rem'>更新</th></tr>
          </thead>
          <tbody>
          {% for c in commands %}
            <tr>
              <td><input type='checkbox' name='names' value='{{ c.name }}' form='bulk-form' class='row-check'></td>
              <td><code>{{ c.name }}</code></td>
              <td>
                <form method='post' action='/guilds/{{ guild_id }}/commands/update'>
                  <input type='hidden' name='csrf' value='{{ csrf }}'>
                  <input type='hidden' name='name' value='{{ c.name }}'>
                  <textarea name='response'>{{ c.response }}</textarea>
                  <details class='muted'>
                    <summary>プレビュー</summary>
                    <p>{{ c.response }}</p>
                  </details>
                  <button type='submit'>更新</button>
                </form>
              </td>
              <td></td>
            </tr>
          {% endfor %}
          </tbody>
        </table>
      </div>

      <form id='bulk-form' method='post' action='/guilds/{{ guild_id }}/commands/bulk-delete'>
        <input type='hidden' name='csrf' value='{{ csrf }}'>
        <button type='submit' class='secondary'>選択を削除</button>
      </form>
    </main>

    <script>
      const all = document.getElementById('select-all');
      if (all) {
        all.addEventListener('change', () => {
          document.querySelectorAll('.row-check').forEach(cb => cb.checked = all.checked);
        });
      }
      // Shared SPA navigation (same as other pages)
      (() => {
        const appSel = '#app';
        const isInternal = (url) => { try { const u = new URL(url, location.href); return u.origin === location.origin; } catch { return false; } };
        const swapContent = async (response, pushUrl) => {
          const html = await response.text();
          const doc = new DOMParser().parseFromString(html, 'text/html');
          const next = doc.querySelector(appSel);
          if (!next) return false;
          const current = document.querySelector(appSel);
          if (!current) return false;
          current.replaceWith(next);
          const t = doc.querySelector('title');
          if (t) document.title = t.textContent || document.title;
          if (pushUrl) history.pushState({}, '', pushUrl);
          return true;
        };
        const navTo = async (url, opts = {}) => {
          try {
            const res = await fetch(url, { credentials: 'include', redirect: 'follow', ...opts, headers: { 'X-Requested-With': 'fetch', ...(opts.headers||{}) } });
            if (!res.ok) { location.href = url; return; }
            const ok = await swapContent(res, opts.method && opts.method !== 'GET' ? res.url : url);
            if (!ok) location.href = url;
          } catch (_) { location.href = url; }
        };
        document.addEventListener('click', (e) => {
          const a = e.target.closest('a');
          if (!a) return;
          if (a.hasAttribute('download') || a.target && a.target !== '' && a.target !== '_self') return;
          const href = a.getAttribute('href');
          if (!href || href.startsWith('#') || href.startsWith('mailto:') || href.startsWith('tel:')) return;
          if (!isInternal(href) || a.dataset.noSpa === 'true') return;
          if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
          e.preventDefault();
          navTo(href);
        });
        document.addEventListener('submit', (e) => {
          const form = e.target;
          if (!(form instanceof HTMLFormElement)) return;
          if (!form.closest(appSel)) return;
          e.preventDefault();
          const method = (form.method || 'GET').toUpperCase();
          const action = form.action || location.href;
          const body = new FormData(form);
          navTo(action, { method, body });
        });
        window.addEventListener('popstate', () => navTo(location.href));
      })();
    </script>
  </body>
</html>
"#, ext = "html" )]
pub struct CommandsTemplate {
    pub guild_id: i64,
    pub q: String,
    pub commands: Vec<CmdRow>,
    pub csrf: String,
}

#[derive(Clone)]
pub struct CmdRow { pub name: String, pub response: String }
