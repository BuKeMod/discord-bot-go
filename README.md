# Personal Discord Bot (Go)

> ต้องการแค่เปิดบอทบน Ubuntu? เริ่มที่ [คู่มือ Ubuntu แบบสั้น](QUICKSTART-UBUNTU.md) ได้เลย ไม่ต้องอ่านคำสั่ง build/test ด้านล่าง

บอทเซิร์ฟเวอร์ส่วนตัว ใช้ DiscordGo + SQLite แบบ pure Go ไม่มี Redis/MySQL/Docker
Go 1.26+ ใช้เฉพาะเครื่องพัฒนา/เครื่อง build; Linux ปลายทางไม่ต้องติดตั้ง Go

## ฟีเจอร์ทั้งหมด (Features)

ทุกคำตอบและปฏิสัมพันธ์ของบอทส่งในรูปแบบ **Discord Embed** สวยงาม มีสีสันระบุสถานะอย่างชัดเจน

| หมวดหมู่ | คำสั่ง / ฟังก์ชัน | การทำงาน |
| --- | --- | --- |
| **🛡️ Moderation** | `/ban user: reason: delete_days:` | แบนสมาชิกออกจากเซิร์ฟเวอร์ พร้อมลบข้อความย้อนหลัง |
| | `/unban user_id: reason:` | ปลดแบนผู้ใช้ด้วย Discord User ID |
| | `/kick user: reason:` | เตะสมาชิกออกจากเซิร์ฟเวอร์ |
| | `/timeout user: minutes: reason:` | ระงับการใช้งานสมาชิก (1 นาที ถึง 28 วัน) |
| | `/untimeout user:` | ยกเลิกการระงับการใช้งาน |
| | `/warn add user: reason:` | ตักเตือนสมาชิกและบันทึกประวัติลงฐานข้อมูล |
| | `/warn list user:` | ดูรายการประวัติการเตือนทั้งหมดของสมาชิก |
| | `/warn remove warn_id:` | ลบประวัติการเตือนด้วย ID |
| | `/purge amount: [user] [contains] [only_bots]` | ลบข้อความจำนวนมากในห้อง (1–100 ข้อความ) พร้อมตัวกรองตามสมาชิก/คำ/บอท |
| **🎮 Party / LFG** | `/party game: [max_players] [time] [note] [expire_minutes]` | สร้างการ์ดนัดเล่นเกมแบบ Interactive (ปุ่มเข้าร่วม/สำรอง/ถอนตัว/ยกเลิก) |
| | **แท็กเตือนเมื่อคนครบ** | บอทจะแท็กสมาชิกตัวจริงทุกคนในห้องอัตโนมัติเมื่อตี้เต็ม |
| | **นับถอยหลัง & หมดเวลา** | ตั้งเวลานัดเล่นและเวลารับสมัคร นับถอยหลังแบบ Live และปิดรับสมัครอัตโนมัติเมื่อหมดเวลา |
| | **ระบบตัวสำรองอัจฉริยะ** | เมื่อตัวจริงถอนตัว ตัวสำรองคิวแรกจะถูกเลื่อนขึ้นเป็นตัวจริงอัตโนมัติ |
| **🎫 Ticket System** | `/ticket panel ...` | ส่ง Embed Panel พร้อมปุ่มกดสร้าง Ticket ในห้องที่ต้องการ |
| | **Ticket Button** | คลิกปุ่มเพื่อสร้างห้องส่วนตัวอัตโนมัติ (เฉพาะผู้เปิดและทีมงาน Support) |
| | `/ticket close reason:` | ปิด Ticket พร้อมปุ่มกดยืนยันป้องกันการกดผิด |
| | **Transcript** | บันทึกประวัติการแชทเป็นไฟล์ HTML (Discord Dark Theme) ส่งเข้าห้องล็อกและ DM |
| **📋 Logs System** | `/logconfig set type: channel:` | ตั้งค่าห้องล็อกแยกตามประเภท (`voice`, `message`, `member`, `mod`, `ticket`, `all`) |
| | **Mod Log** | ดักจับการ Ban, Unban, Kick, Timeout จาก Audit Log อัตโนมัติ |
| | **Message Log** | บันทึกการลบข้อความ และการแก้ไขข้อความ (ก่อนแก้ / หลังแก้) |
| | **Member Log** | บันทึกการเข้า/ออกจากเซิร์ฟเวอร์, การเปลี่ยนชื่อเล่น, และการเปลี่ยน Role |
| | **Voice Log** | บันทึกการเข้า, ออก และย้ายห้องเสียง |
| **🎙️ Voice Activity** | `/voiceactivity user:` | ดูเวลาสะสมทั้งหมดที่อยู่ในห้องเสียง และสถานะปัจจุบัน |
| | `/voicetop` | แสดง 10 อันดับสมาชิกที่อยู่ในห้องเสียงนานที่สุดในเซิร์ฟเวอร์ |
| **⚙️ Utility** | `/status` | ตรวจสอบสถานะระบบ CPU, RAM (Alloc/Sys), Uptime, Latency, Goroutines, DB สำหรับแอดมิน |
| | `/avatar user:` | ดูรูปโปรไฟล์ขนาดเต็มพร้อมลิงก์ดาวน์โหลดใน Embed |
| | `/userinfo user:` | ดูวันสร้างบัญชี วันเข้าเซิร์ฟเวอร์ และรายการ Roles พร้อมรูป Thumbnail |
| | `/serverinfo` | ดูข้อมูลเซิร์ฟเวอร์ สมาชิก ออนไลน์ บูสต์ หมวดหมู่ และห้อง |
| | `/roleinfo role:` | ดูข้อมูล Role, รหัสสี Hex, ลำดับ และสิทธิ์สำคัญ |
| | `/poll question: options: ...` | สร้างโพลสำรวจความคิดเห็น |
| | `/remind in: reason:` | ตั้งเวลาแจ้งเตือนความจำ |
| | `/timestamp format: ...` | แปลงเวลาเป็น Discord timestamp |
| | `/ping` | ตรวจสอบความหน่วง Gateway heartbeat และสถานะ |
| | `/help` | รายการคำสั่งทั้งหมดแยกหมวดหมู่อย่างสวยงาม |
| **🎲 Fun** | `/roll`, `/coinflip`, `/choose` | ทอยเต๋า โยนหัวก้อย และสุ่มตัวเลือก |
| **🎭 Welcome & Roles** | `/welcome`, `/role`, `/roleconfig` | ระบบต้อนรับสมาชิกใหม่ และระบบเลือกยศด้วยตนเอง |


## 📋 ลำดับขั้นตอนการตั้งค่าและเปิดระบบ (Step-by-Step Setup Guide)

### ขั้นตอนที่ 1: เตรียม Bot บน Discord Developer Portal
1. เข้าไปที่ [Discord Developer Portal](https://discord.com/developers/applications) → กดปุ่ม **New Application** เพื่อสร้างแอปพลิเคชัน
2. ไปที่เมนู **Bot** (แถบด้านซ้าย):
   - กดปุ่ม **Reset Token** แล้วคัดลอก Token เก็บไว้ (ห้ามเผยแพร่)
   - ในหัวข้อ **Privileged Gateway Intents**:
     - ✅ ติ๊กเปิด **Server Members Intent** (จำเป็นสำหรับระบบ Welcome, Member Logs และการตรวจจับสมาชิก)
3. ไปที่เมนู **OAuth2** → **URL Generator**:
   - ในช่อง **Scopes**: ติ๊ก `bot` และ `applications.commands`
   - ในช่อง **Bot Permissions**: ติ๊ก `Administrator` (หรือสิทธิ์ View Channels, Send Messages, Manage Roles ตามต้องการ)
   - คัดลอกลิงก์ด้านล่างสุด ไปเปิดในเบราว์เซอร์เพื่อ **เชิญบอทเข้าสู่เซิร์ฟเวอร์ Discord ของคุณ**

---

### ขั้นตอนที่ 2: ตั้งค่าคอนฟิก `.env` บนเซิร์ฟเวอร์ Ubuntu
1. เข้าไปในโฟลเดอร์โปรเจกต์ของบอท:
   ```sh
   cd discord-bot
   ```
2. สร้างไฟล์ `.env` จากตัวอย่าง:
   ```sh
   cp .env.example .env
   nano .env
   ```
3. กำหนดค่าต่าง ๆ ในไฟล์ `.env`:
   ```env
   # โทเคนบอทที่ได้จาก Developer Portal
   DISCORD_TOKEN=your_bot_token_here

   # Server ID หลัก (เปิด Developer Mode ใน Discord -> คลิกขวาชื่อเซิร์ฟเวอร์ -> Copy Server ID)
   DISCORD_GUILD_ID=your_server_id_here

   # อนุญาตให้บอททำงานได้หลายเซิร์ฟเวอร์พร้อมกัน (แยกข้อมูลแต่ละเซิร์ฟเวอร์เด็ดขาด)
   MULTI_GUILD=true

   # ลงทะเบียนคำสั่ง Slash Commands ระดับ Global (โผล่ทุกเซิร์ฟเวอร์และมีปุ่มบนโปรไฟล์บอท)
   GLOBAL_COMMANDS=true

   # เปิดระบบต้อนรับสมาชิกใหม่เข้าห้อง
   WELCOME_ENABLED=true

   # ข้อความสถานะกิจกรรมของบอท
   BOT_STATUS=/help
   ```
   *(กด `Ctrl+O` แล้ว `Enter` เพื่อบันทึก และ `Ctrl+X` เพื่อออกจาก nano)*
4. จำกัดสิทธิ์ความปลอดภัยของไฟล์คอนฟิก:
   ```sh
   chmod 600 .env
   ```

---

### ขั้นตอนที่ 3: ติดตั้งและเปิดระบบให้รันอัตโนมัติ (Systemd Service)

#### กรณีติดตั้งครั้งแรกสุด (First-time Installation):
รันคำสั่งติดตั้งอัตโนมัติ:
```sh
./scripts/install.sh
```
> ตัวติดตั้งจะทำการ:
> 1. ตรวจสอบสถาปัตยกรรม CPU (amd64 / arm64) ให้อัตโนมัติ
> 2. สร้าง System User `discord-bot` และโฟลเดอร์เก็บฐานข้อมูล `/var/lib/discord-bot`
> 3. คัดลอกคอนฟิกไปยัง `/etc/discord-bot.env` (โดยจัดการตำแหน่ง DB ให้ปลอดภัย)
> 4. ลงทะเบียนคำสั่ง Slash Commands ไปยัง Discord
> 5. เปิด Service ทำงานในเบื้องหลัง 24 ชม. และเริ่มทำงานใหม่อัตโนมัติเมื่อเซิร์ฟเวอร์ Reboot

#### กรณีอัปเดตไฟล์ / เปลี่ยน Token / แก้ไข .env ภายหลัง:
เมื่อมีการแก้ไขค่าใน `.env` หรืออัปเดตโค้ด เพียงรัน:
```sh
./scripts/update.sh
```
> สคริปต์ `update.sh` จะทำการ:
> 1. Sync การตั้งค่าจาก `.env` ไปยัง `/etc/discord-bot.env` ให้อัตโนมัติ
> 2. อัปเดตตัวโปรแกรมบอทเข้าสู่ service
> 3. รีสตาร์ตบอทใหม่และลงทะเบียนคำสั่งให้อัตโนมัติทันที

---

### ขั้นตอนที่ 4: คำสั่งสำหรับตรวจสอบและควบคุมระบบ (Cheat Sheet)

| การทำงาน | คำสั่งที่ใช้ |
| :--- | :--- |
| **ตรวจสอบสถานะบอท** | `sudo systemctl status discord-bot` |
| **ดู Log การทำงานแบบเรียลไทม์** | `sudo journalctl -u discord-bot -f` |
| **ดู Log ย้อนหลัง 30 บรรทัดล่าสุด** | `sudo journalctl -u discord-bot -n 30 --no-pager` |
| **รีสตาร์ตบอท** | `sudo systemctl restart discord-bot` |
| **หยุดการทำงานของบอท** | `sudo systemctl stop discord-bot` |
| **เปิดการทำงานของบอท** | `sudo systemctl start discord-bot` |
| **ดูรายการคำสั่ง Slash Commands บน Discord** | `./bin/discord-bot -list-commands` |
| **ลงทะเบียนคำสั่ง Global ด้วยตนเอง** | `./bin/discord-bot -register -global` |
| **ล้างคำสั่งขยะที่ตกค้างบน Discord ทั้งหมด** | `./bin/discord-bot -clear-all` |

---

### ขั้นตอนที่ 5: การแก้ปัญหาที่พบบ่อย (Troubleshooting & FAQs)

1. **Error: `mkdir data: read-only file system`**:
   - **สาเหตุ**: เกิดจากไฟล์ `/etc/discord-bot.env` มีการระบุ `DB_PATH=data/bot.db` ซึ่งขัดกับนโยบายความปลอดภัยของ Systemd Service
   - **วิธีแก้**: รันคำสั่งนี้เพื่อเปลี่ยนตำแหน่งฐานข้อมูลให้ถูกต้อง:
     ```sh
     sudo sed -i 's|^DB_PATH=.*|DB_PATH=/var/lib/discord-bot/bot.db|' /etc/discord-bot.env
     sudo systemctl restart discord-bot
     ```

2. **เปลี่ยน Token แล้ว แต่ทำไมใน Discord ยังเป็นบอทตัวเดิม?**:
   - **สาเหตุ 1**: กดปุ่ม *Reset Token* ใน Developer Portal ของแอปเดิม (ชื่อบอทและ ID ยังคงเป็นตัวเดิม แค่เปลี่ยนรหัส) หากต้องการบอทตัวใหม่ต้องกด *New Application*
   - **สาเหตุ 2**: สร้างบอทใหม่แล้วแต่ยังไม่ได้เชิญเข้าเซิร์ฟเวอร์ ให้ตรวจสอบว่ามีบอทตัวใหม่อยู่ในรายชื่อสมาชิกเซิร์ฟเวอร์แล้วหรือไม่
   - **สาเหตุ 3**: คอนฟิกในระบบยังไม่ได้อัปเดต ให้รัน `./scripts/update.sh` เพื่อ sync คอนฟิกเข้า service

3. **รันคำสั่งแล้วขึ้น `command not found`**:
   - สามารถเรียกใช้งานผ่าน `./bin/discord-bot` หรือ `./bin/discord-bot-linux-amd64` ได้ทั้งสองแบบ

## การใช้งานหลายเซิร์ฟเวอร์ (Multi-Guild Mode)

หากต้องการให้บอทสามารถเข้าร่วมและให้บริการได้หลายเซิร์ฟเวอร์:
1. ตั้งค่าใน `/etc/discord-bot.env`:
   ```env
   MULTI_GUILD=true
   GLOBAL_COMMANDS=true
   ```
2. รันอัปเดตหรือ restart บอท:
   - บอทจะลงทะเบียนเป็น Global Commands อัตโนมัติ ทำให้คำสั่งไปโผล่ในทุกเซิร์ฟเวอร์ที่บอทเข้าร่วม
   - ข้อมูลของแต่ละเซิร์ฟเวอร์ (ห้องล็อก, ตั๋ว Ticket, ประวัติ Warn, Voice Activity, Party, Welcome) จะถูกแยกจากกันโดยเด็ดขาดผ่านคอลัมน์ `guild` ในฐานข้อมูล SQLite
   - สมาชิกสามารถกดปุ่ม **`+ Add App`** บนหน้าโปรไฟล์บอทเพื่อเชิญบอทไปใช้งานในเซิร์ฟเวอร์ของตนเองได้ทันที

## การจัดการโมดูล (Admin Feature Toggle - /module)

ผู้ดูแลเซิร์ฟเวอร์ (Administrator) สามารถเปิดหรือปิดการทำงานของแต่ละโมดูลในเซิร์ฟเวอร์นั้น ๆ ได้อย่างอิสระผ่านคำสั่ง `/module`:
- หน้าต่างควบคุมเป็น **Interactive Control Panel** ที่มีทั้ง Dropdown Select Menu และ Action Buttons
- **Select Menu (ดรอปดาวน์)**: แสดงสถานะโมดูลแบบเรียลไทม์ (`[เปิดอยู่ ➔ คลิกเพื่อปิด]` หรือ `[ปิดอยู่ ➔ คลิกเพื่อเปิด]`) เมื่อคลิกเลือกตัวใดจะสลับสถานะทันทีโดยไม่ต้องพิมพ์คำสั่งใหม่
- **Action Buttons**:
  - `🟢 เปิดใช้งานทั้งหมด`: เปิดทุกโมดูลในเซิร์ฟเวอร์
  - `🔴 ปิดใช้งานทั้งหมด`: ปิดทุกโมดูลในเซิร์ฟเวอร์
  - `🔄 รีเฟรชสถานะ`: อัปเดตข้อมูลบนหน้าจอแผงควบคุม
- **ผลของการปิดโมดูล**:
  - **คำสั่ง Slash Command**: เมื่อสมาชิกพยายามใช้คำสั่งในโมดูลที่ปิด จะได้รับข้อความแจ้งเตือนสีส้มแบบ Ephemeral ทันที
  - **Interactive Components**: ปุ่มกดต่าง ๆ เช่น ปุ่มตั๋ว หรือปุ่มตี้เกม จะถูกบล็อกไม่ให้ทำงาน
  - **Background Event Hooks**: บอทจะไม่ประมวลผล event ของโมดูลนั้น เช่น Message Delete Log, Voice Log, Audit Mod Log, Welcome เมื่อโมดูลถูกปิดในเซิร์ฟเวอร์นั้น
  - **ความปลอดภัย**: คำสั่งหลักของระบบ เช่น `/module`, `/status`, `/help`, `/ping` จะไม่มีวันถูกปิด เพื่อป้องกันไม่ให้แอดมินล็อคตัวเองออกจากระบบ

## ความปลอดภัยของ role

ค่าเริ่มต้น `ROLES_ENABLED=false`: ไม่ลงทะเบียน /role และ /roleconfig และ handler ปฏิเสธคำสั่ง role เก่าด้วย
หากต้องการเปิด ตั้ง `ROLES_ENABLED=true` แล้วรัน `-register` และ restart บอท

เป็น self-service เท่านั้น ไม่มีพารามิเตอร์เลือกผู้ใช้คนอื่น
ตรวจ allowlist, สิทธิ์ Manage Roles ของบอท, managed role และลำดับ role ทุกครั้งผ่าน REST
รับเฉพาะ role ที่มี server permission เป็นศูนย์ รวมถึงปฏิเสธ permission bit ที่ไม่รู้จัก
ปฏิเสธ @everyone และ role ที่สูงเท่ากับ/สูงกว่าบอท
Administrator เท่านั้นที่แก้ allowlist/welcome ได้ และตรวจสิทธิ์ซ้ำใน handler

**ผู้ดูแลต้องตรวจ channel overwrites ก่อนอนุญาต role:** role ที่ไม่มี server permissions
ยังอาจให้เข้าถึงห้องลับผ่านสิทธิ์เฉพาะห้องได้ จึงควรใช้ cosmetic/opt-in roles ที่ตั้งใจให้ทุกคนเลือก
ถ้าเปลี่ยน permissions ของ role ภายหลัง บอทจะปฏิเสธทั้ง add/remove ให้ผู้ดูแลจัดการเอง
`deny` ปิด self-service แต่ไม่ได้ถอด role จากสมาชิกเดิม
การตรวจและแก้ role เป็นคนละ Discord API request จึงไม่ใช่ transaction กับการแก้ role โดยผู้ดูแล

## การเพิ่มฟีเจอร์และอัปเดต

อ่าน [FEATURES.md](FEATURES.md): แยกโฟลเดอร์ฟีเจอร์และใช้ `./scripts/update.sh` เพื่อ build/restart ด้วยคำสั่งเดียว

## โครงสร้าง

```text
main.go                         # lifecycle, intents, shutdown
internal/config/config.go       # .env / validation
features/                       # ping, help, serverinfo, welcome, roles
internal/feature/               # module contract, registry, permission checks
internal/commands/              # interaction routing, registration
internal/storage/storage.go     # SQLite config persistence
deploy/discord-bot.service
.env.example
go.mod / go.sum
```

SQLite มี connection เดียว, WAL และ busy timeout 5 วินาที
ปิดโปรเซสด้วย SIGTERM/Ctrl+C: หยุดรับงานใหม่ รอ handler จบ แล้วปิด Gateway และฐานข้อมูล
HTTP timeout 15 วินาที และ context จำกัดเวลางาน REST/DB
ไม่เก็บ token ใน log โดยตั้งใจ; ควรจำกัดคนอ่าน log เพราะมี user/role/server IDs และ API errors

## Test และ Build

```sh
go generate ./features
go test ./...
go vet ./...
mkdir -p bin
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags='-s -w' -o bin/discord-bot-linux-amd64 .
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags='-s -w' -o bin/discord-bot-linux-arm64 .
```

ใช้ modernc.org/sqlite จึง cross-compile ได้โดยไม่ใช้ C compiler หรือ SQLite shared library
Linux ต้องมี CA certificates สำหรับ HTTPS และเข้าถึง Discord ผ่านอินเทอร์เน็ต
ไม่มีการอ้างตัวเลข RAM ตายตัว: ขึ้นกับ guild metadata, จำนวน event และ Go GC
ใช้ `systemctl status`/เครื่องมือระบบวัดจริงก่อนกำหนด memory limit

## Linux + systemd

ตัวอย่างสำหรับเครื่องที่มี systemd ใช้ user เฉพาะและติดตั้ง binary ที่ตรง CPU:

```sh
sudo useradd --system --home /opt/discord-bot --shell /usr/sbin/nologin discord-bot
sudo install -d -m 755 /opt/discord-bot
sudo install -m 755 bin/discord-bot-linux-amd64 /opt/discord-bot/discord-bot
sudo install -m 600 .env.example /etc/discord-bot.env
sudoedit /etc/discord-bot.env
# ตั้ง token/guild และ WELCOME_ENABLED; ลบ DB_PATH ออกจากไฟล์นี้
# เพื่อใช้ /var/lib/discord-bot/bot.db จาก service
sudo install -m 644 deploy/discord-bot.service /etc/systemd/system/discord-bot.service
sudo systemctl daemon-reload
# ลงทะเบียนจากเครื่องพัฒนาด้วย .env ที่ชี้ guild เดียวกันก่อนเปิด service
go run . -register
sudo systemctl enable --now discord-bot
sudo journalctl -u discord-bot -f
```

อย่าวาง `.env` เพิ่มใน `/opt/discord-bot`; service อ่าน `/etc/discord-bot.env`
ตั้ง DB_PATH เป็น `/var/lib/discord-bot/bot.db` ถ้าระบุไว้ใน EnvironmentFile เพราะมีลำดับสูงกว่า Environment
systemd สร้าง data directory ให้เอง; เก็บ binary/config เป็นของ root และให้บอทเขียนได้เฉพาะ data
อัปเดต binary: stop service → install binary ใหม่ → start service
สำรอง SQLite แบบง่าย: stop service → copy data directory รวม WAL/SHM ถ้ามี → start service

## ตรวจรับกับเซิร์ฟเวอร์จริง

หลังใส่ token ให้ลอง /ping, /help, /serverinfo แล้วตรวจว่า non-admin ใช้ /roleconfig ไม่ได้
ทดสอบ role ที่อนุญาต/ไม่อนุญาต/managed/มี permission/อยู่เหนือบอท และตรวจ add/remove ด้วยสมาชิกปกติ
เปิด welcome แล้วให้บัญชีทดสอบเข้าเซิร์ฟเวอร์ ตรวจข้อความ/mentions และลอง restart ให้ config ยังอยู่
ส่ง SIGTERM แล้วตรวจ log shutdown; ทดสอบ reconnect โดยตัดเน็ตชั่วคราว
การทดสอบอัตโนมัติไม่ต้องใช้ token; การเชื่อมต่อ Discord จริงต้องทำกับ application ของคุณ

## เอกสารต้นทาง

- [DiscordGo](https://github.com/bwmarrin/discordgo)
- [Discord Gateway intents](https://github.com/discord/discord-api-docs/blob/main/developers/events/gateway.mdx)
- [Discord permissions](https://github.com/discord/discord-api-docs/blob/main/developers/topics/permissions.mdx)
- [Pure Go SQLite](https://pkg.go.dev/modernc.org/sqlite)
