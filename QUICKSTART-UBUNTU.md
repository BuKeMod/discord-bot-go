# เริ่มรันบน Ubuntu แบบสั้น

ทำตามหน้านี้หน้าเดียวเพื่อทดสอบบอทครั้งแรก ไม่ต้องติดตั้ง Go, Docker หรือ SQLite บน Ubuntu

## 1. เตรียม Discord

- ใน [Discord Developer Portal](https://discord.com/developers/applications) สร้าง Application แล้วเพิ่ม Bot
- กด Reset Token/Copy Token แล้วเก็บไว้ ห้ามส่ง token ให้ใคร
- คัดลอก **Server ID**: เปิด Developer Mode ใน Discord → คลิกขวาชื่อเซิร์ฟเวอร์ → Copy Server ID
- OAuth2 → URL Generator เลือก scopes `bot` และ `applications.commands`
- เลือก permissions `View Channels` และ `Send Messages` แล้วเปิดลิงก์เชิญบอทเข้าเซิร์ฟเวอร์
- ยังไม่ต้องเปิด intent พิเศษ; welcome ปิดไว้ก่อน

## 2. แตกไฟล์และเลือกโปรแกรมให้ตรงเครื่อง

อัปโหลด `discord-bot.zip` ไป Ubuntu แล้วรัน:

```sh
sudo apt update && sudo apt install -y unzip
unzip discord-bot.zip
cd discord-bot
uname -m
```

ใช้ไฟล์ตามผล `uname -m`:

- `x86_64` → `bin/discord-bot-linux-amd64`
- `aarch64` → `bin/discord-bot-linux-arm64`

ตัวอย่างด้านล่างใช้ `amd64` ถ้าเครื่องแสดง `aarch64` ให้แทนชื่อไฟล์เป็น `arm64` ทุกจุด

## 3. ใส่ token และ Server ID

```sh
cp .env.example .env
nano .env
```

กรอกสองค่านี้ แล้วบันทึกใน nano ด้วย `Ctrl+O`, Enter, `Ctrl+X`:

```env
DISCORD_TOKEN=วางโทเคนบอทตรงนี้
DISCORD_GUILD_ID=วาง_SERVER_ID_ตรงนี้
```

อย่าใส่เครื่องหมาย quote ครอบค่า token; ปล่อย `WELCOME_ENABLED=false` ไว้ก่อน แล้วจำกัดสิทธิ์ไฟล์:

```sh
chmod 600 .env
```

## 4. ลงทะเบียนคำสั่งและเปิดบอท

```sh
./bin/discord-bot -register -global
./bin/discord-bot
```

บรรทัดแรกแทนที่คำสั่งของแอปนี้ด้วยคำสั่งทั้งหมด 27 คำสั่งระดับ Global
บรรทัดที่สองเปิดบอทเพื่อทดสอบ หน้าต่างนี้ต้องเปิดค้างไว้; กด `Ctrl+C` เพื่อปิด

ลองพิมพ์ `/ping` หรือ `/help` ใน Discord ได้เลยครับ

## 5. ทำงานเบื้องหลัง 24 ชม. และเริ่มเองหลัง reboot (Systemd)

หยุดบอทที่เปิดค้างใน terminal ด้วย Ctrl+C แล้วรันจากโฟลเดอร์โปรเจกต์:

```sh
./scripts/install.sh
```

ติดตั้งครั้งเดียว จากนั้นปิดหน้าจอ SSH ได้ บอทจะออนไลน์ตลอด 24 ชม.
เมื่อใดก็ตามที่มีการแก้ `.env` หรืออัปเดตโค้ด แค่รันคำสั่งเดียว:

```sh
./scripts/update.sh
```

คำสั่งนี้จะ Sync คอนฟิก `.env` เข้า Service, อัปเดตโปรแกรม, และรีสตาร์ตบอทให้อัตโนมัติทันที

## ถ้ามีปัญหา

- `mkdir data: read-only file system`: รัน `sudo sed -i 's|^DB_PATH=.*|DB_PATH=/var/lib/discord-bot/bot.db|' /etc/discord-bot.env && sudo systemctl restart discord-bot`
- `401 Unauthorized`: token ผิดหรือถูก reset; แก้ `.env` แล้วรัน `./scripts/update.sh`
- ตรวจสอบสถานะ: `sudo systemctl status discord-bot`
- ดู Log สด: `sudo journalctl -u discord-bot -f`
