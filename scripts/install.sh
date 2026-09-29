#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $(uname -s) == Linux ]] || { echo 'Run install.sh on Ubuntu.' >&2; exit 1; }
case "$(uname -m)" in x86_64) target_arch=amd64;; aarch64) target_arch=arm64;; *) echo 'Unsupported CPU' >&2;exit 1;; esac
binary="bin/discord-bot-linux-$target_arch"
[[ -f "$binary" ]] || { echo 'Build a Linux binary first.' >&2; exit 1; }
sudo -v
if sudo test -f /etc/systemd/system/discord-bot.service; then
 echo 'Service already installed. Use ./scripts/update.sh with a binary path.' >&2; exit 1
fi
[[ -f .env ]] || { echo 'Create .env with token and guild ID first.' >&2; exit 1; }
# Stop a manually started bot with Ctrl+C before running this installer.
if ! id discord-bot >/dev/null 2>&1; then
 sudo useradd --system --home /opt/discord-bot --shell /usr/sbin/nologin discord-bot
fi
sudo install -d -m 755 /opt/discord-bot
sudo install -d -m 700 -o discord-bot -g discord-bot /var/lib/discord-bot
sudo install -m 755 "$binary" /opt/discord-bot/discord-bot
if ! sudo test -e /etc/discord-bot.env; then
 env_temp=$(mktemp)
 trap 'rm -f "$env_temp"' EXIT
 # The service owns DB_PATH; retain all other user settings without evaluating them.
 awk '!/^[[:space:]]*(export[[:space:]]+)?DB_PATH[[:space:]]*=/' .env > "$env_temp"
 sudo install -m 600 "$env_temp" /etc/discord-bot.env
fi
if [[ -f data/bot.db ]] && ! sudo test -e /var/lib/discord-bot/bot.db; then
 for db_file in data/bot.db data/bot.db-wal data/bot.db-shm; do
  if [[ -f "$db_file" ]]; then sudo install -m 600 -o discord-bot -g discord-bot "$db_file" /var/lib/discord-bot/; fi
 done
fi
sudo install -m 644 deploy/discord-bot.service /etc/systemd/system/discord-bot.service
sudo systemctl daemon-reload
sudo systemctl enable --now discord-bot
sudo systemctl --no-pager status discord-bot
