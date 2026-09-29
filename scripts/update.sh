#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
[[ $(uname -s) == Linux ]] || { echo 'Run update.sh on the Ubuntu server.' >&2; exit 1; }
case "$(uname -m)" in x86_64) target_arch=amd64;; aarch64) target_arch=arm64;; *) echo 'Unsupported CPU' >&2;exit 1;; esac
if [[ $# -eq 0 ]] || [[ "${1:-}" == "bin/discord-bot" ]] || [[ "${1:-}" == "discord-bot" ]]; then
  if ! command -v go >/dev/null && [[ -f "bin/discord-bot-linux-$target_arch" ]]; then
    echo "Using pre-built bin/discord-bot-linux-$target_arch"
    binary="bin/discord-bot-linux-$target_arch"
  else
    GOOS=linux GOARCH="$target_arch" ./scripts/build.sh
    binary="bin/discord-bot-linux-$target_arch"
  fi
else
  binary="$1"
fi
[[ -f "$binary" ]] || { echo "Binary not found: $binary" >&2; exit 1; }
sudo -v
sudo test -f /opt/discord-bot/discord-bot || { echo 'Run scripts/install.sh first.' >&2; exit 1; }
sudo test -f /etc/discord-bot.env

# Automatically sync .env to /etc/discord-bot.env if .env exists
if [[ -f .env ]]; then
  echo "Syncing .env to /etc/discord-bot.env..."
  env_temp=$(mktemp)
  awk '!/^[[:space:]]*(export[[:space:]]+)?DB_PATH[[:space:]]*=/' .env > "$env_temp"
  sudo install -m 600 "$env_temp" /etc/discord-bot.env
  rm -f "$env_temp"
fi
# Take a lock before deployment. Build errors above do not interrupt the live bot.
exec 9>/tmp/discord-bot-update.lock
flock -n 9 || { echo 'Another update is running.' >&2; exit 1; }
sudo install -m 755 "$binary" /opt/discord-bot/discord-bot.next
sudo cp -p /opt/discord-bot/discord-bot /opt/discord-bot/discord-bot.previous
sudo cp -p /etc/systemd/system/discord-bot.service /opt/discord-bot/service.previous
sudo systemctl stop discord-bot
sudo mv /opt/discord-bot/discord-bot.next /opt/discord-bot/discord-bot
sudo install -m 644 deploy/discord-bot.service /etc/systemd/system/discord-bot.service
sudo systemctl daemon-reload
if ! sudo systemctl restart discord-bot; then
  echo 'Startup/registration failed. Restoring previous program and service.' >&2
  sudo cp -p /opt/discord-bot/discord-bot.previous /opt/discord-bot/discord-bot
  sudo cp -p /opt/discord-bot/service.previous /etc/systemd/system/discord-bot.service
  sudo systemctl daemon-reload
  sudo systemctl restart discord-bot || true
  echo 'Inspect: sudo journalctl -u discord-bot -n 50' >&2
  exit 1
fi
sudo systemctl --no-pager status discord-bot
