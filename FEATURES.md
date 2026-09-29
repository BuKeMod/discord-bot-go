# เพิ่มฟีเจอร์และอัปเดต Ubuntu

ฟีเจอร์แต่ละตัวอยู่ในโฟลเดอร์ของตัวเอง:

```text
features/
  Admin_module/            ← /module ควบคุม เปิด/ปิด แต่ละโมดูลแบบแยกเซิร์ฟเวอร์ (Interactive UI)
  Admin_status/            ← /status ดูสถานะระบบ CPU, RAM, Uptime สำหรับแอดมิน
  ping/                    ← เช็ค latency
  help/                    ← แสดงคำสั่งแยกหมวดหมู่อย่างสวยงาม
  welcome/                 ← ระบบต้อนรับสมาชิกใหม่
  roles/                   ← ระบบ Self-service Role
  Mod_actions/             ← /ban, /unban, /kick, /timeout, /untimeout, /warn, /purge
  Logs_mod/                ← Audit log จับ Ban/Kick/Timeout ส่งห้อง mod log
  Utility_ticket/          ← ระบบ Ticket, ปุ่มกดสร้างห้อง, HTML Transcript
  Utility_party/           ← ระบบตั้งตี้เล่นเกม Interactive ปุ่มเข้าร่วม/สำรอง นับถอยหลัง แท็กเตือนเมื่อครบ
  Logs_config/             ← /logconfig ตั้งค่าช่องรับ logs แต่ละประเภท
  Logs_message/            ← บันทึกการลบและแก้ไขข้อความ
  Logs_member/             ← บันทึกเข้า/ออก/เปลี่ยนชื่อ/ยศ
  Voice_log/               ← บันทึกเข้า/ออก/ย้ายห้องเสียง
  Voice_activity/          ← บันทึกเวลาอยู่ในห้องเสียง /voiceactivity, /voicetop
  Utility_avatar/          ← รูปโปรไฟล์ขนาดเต็มใน Embed
  Utility_userinfo/        ← ข้อมูลสมาชิก บัญชี ยศ
  Utility_serverinfo/      ← ข้อมูลเซิร์ฟเวอร์ จำนวนสมาชิก ห้อง บูสต์
  Utility_roleinfo/        ← ข้อมูล Role และสิทธิ์สำคัญ
  Utility_poll/            ← สร้างโพลสำรวจความคิดเห็น
  Utility_remind/          ← ตั้งเวลาเตือนความจำ
  Utility_timestamp/       ← แปลงเวลาเป็น Discord timestamp
  Fun_roll/                ← ทอยลูกเต๋า
  Fun_coinflip/            ← โยนหัวก้อย
  Fun_choose/              ← สุ่มเลือกคำตอบ
  catalog_generated.go     ← ระบบสร้างให้อัตโนมัติ
internal/
  feature/                 ← รองรับ Module, ComponentHandler, Event hooks, Embeds
  featureutil/             ← BotHasChannelPermissions แก้บั๊กสิทธิ์ 403
  commands/                ← interaction router, BuildEmbed แปลงข้อความเป็น Embed
  config/                  ← อ่านการตั้งค่าจากสภาพแวดล้อม
  storage/                 ← SQLite จัดการ warns, tickets, voice_activity, logs
scripts/
  build.sh
  install.sh
  update.sh                ← อัปเดตบอท (ใช้ binary สำเร็จรูปได้โดยไม่ต้องลง Go บนเซิร์ฟเวอร์)

```

## ครั้งแรก: ย้ายจากการเปิดค้างใน terminal ไป systemd

1. เก็บโค้ดเวอร์ชันนี้ไว้ในโฟลเดอร์งานบน Ubuntu พร้อม `.env` เดิมและ `data/` เดิม
2. หยุดบอทที่รันใน terminal ด้วย Ctrl+C ก่อน
3. รัน `./scripts/install.sh`

ใช้ binary ที่เตรียมไว้ได้ ไม่ต้องลง Go สำหรับการติดตั้งครั้งนี้
ตัวติดตั้งเลือก amd64/arm64 เอง สร้าง service และเปิดบอทอัตโนมัติหลัง reboot
ถ้ามี service อยู่แล้ว ให้ใช้ `./scripts/update.sh bin/discord-bot-linux-amd64` (หรือ arm64)

ตัวติดตั้งย้ายข้อมูลจาก **ค่า default `data/bot.db`** ไป `/var/lib/discord-bot/` หากปลายทางยังไม่มีฐานข้อมูล
ถ้าคุณตั้ง DB_PATH ไว้ที่อื่น ต้องหยุดบอทแล้วสำรอง/ย้ายฐานข้อมูลนั้นก่อนติดตั้ง
ไฟล์ token และ config ถูกเก็บใน `/etc/discord-bot.env`; ฐานข้อมูลอยู่ `/var/lib/discord-bot/bot.db`
ตัวติดตั้งไม่เขียนทับ config หรือฐานข้อมูลที่มีอยู่แล้ว; หากมี `/etc/discord-bot.env` เดิม ให้ตรวจ DB_PATH ว่าชี้ที่ `/var/lib/discord-bot/bot.db`

## เพิ่มฟีเจอร์ใหม่

```sh
cp -R examples/hello features/hello
./scripts/update.sh
```

เมื่อทำบน Ubuntu ที่ติดตั้ง service แล้ว คำสั่ง update จะ:

1. ค้นหา `features/*/feature.go` และสร้างรายการ imports ให้อัตโนมัติ
2. build โปรแกรมให้เสร็จก่อนหยุดบอท
3. เก็บ binary เดิมไว้เป็น `.previous` แล้วเปลี่ยนโปรแกรม
4. restart service ซึ่งลงทะเบียนคำสั่งชุดปัจจุบันก่อนเปิดบอท

เครื่องที่ build ต้องมี **Go 1.26 ขึ้นไป** และเข้าถึง dependencies ในครั้งแรก
ไม่ต้องแก้ main.go หรือ catalog_generated.go เมื่อเพิ่มโฟลเดอร์
Go ต้อง build/restart มีช่วงบอทหยุดสั้น ๆ ไม่ใช่ hot reload
เวลาอัปเดตไม่ต้องลบทั้งโฟลเดอร์หรือคัดลอก `.env` / ฐานข้อมูลทับใหม่

ถ้าไม่อยากติดตั้ง Go บน Ubuntu: build บนเครื่องพัฒนาแทน เช่น

```sh
GOOS=linux GOARCH=amd64 ./scripts/build.sh
```

อัปโหลด binary ใหม่ แล้วบน Ubuntu รัน:

```sh
./scripts/update.sh bin/discord-bot-linux-amd64
```

## สัญญาของฟีเจอร์

โฟลเดอร์ต้องเป็นชื่ออังกฤษตัวเล็ก เช่น `hello` หรือ `daily_quote`
ต้องมี `feature.go` ที่ export `New(d feature.Dependencies) feature.Module`
`New` ควรสร้าง definitions/handlers เท่านั้น: ไม่เรียก Discord หรืออ่านฐานข้อมูลระหว่างสร้าง
เพราะขั้นตอนลงทะเบียนคำสั่งยังไม่เปิดฐานข้อมูลให้ ใช้ `d.Store` ภายใน handler

- `Commands`: รายการ slash command พร้อม handler
- `Intents`: เพิ่มเฉพาะ intent ที่ฟีเจอร์ต้องใช้; privileged intents ต้องเปิดใน Portal ด้วย
- `MemberAdd`: optional hook เมื่อสมาชิกเข้าร่วม (event อื่นเพิ่มผ่าน core ก่อน)
- `DefaultMemberPermissions`: core ตรวจสิทธิ์ซ้ำก่อนเรียก handler รวมถึง Administrator
- ใช้ `context` ที่ได้รับกับ REST/SQLite เพื่อให้งานจบได้ขณะ shutdown
- ส่งข้อความกลับผ่านค่าคืนของ handler; core ตอบแบบ ephemeral และปิด mentions

ชื่อฟีเจอร์และคำสั่งห้ามซ้ำ ระบบจะปฏิเสธแทนการทับ handler
ฟีเจอร์เป็นโค้ด Go ที่เชื่อถือได้ มีสิทธิ์เท่ากับโปรแกรม ไม่ใช่ sandbox สำหรับโค้ดจากคนแปลกหน้า
`/help` อ่านรายการคำสั่งที่ลงทะเบียนไว้ จึงรวมฟีเจอร์ใหม่ให้เองหลังอัปเดต
role ยังคงปิดโดย default; เปิดด้วย `ROLES_ENABLED=true` ใน config ของ service แล้ว restart

## คำสั่งดูแล

```sh
sudo systemctl status discord-bot
sudo systemctl restart discord-bot
sudo systemctl stop discord-bot
sudo journalctl -u discord-bot -f
```

เมื่อแก้ config ของ service ใช้ `sudoedit /etc/discord-bot.env` แล้ว restart
ทุก start/restart จะ sync เฉพาะ guild commands ของแอปนี้; ไม่ลบ global commands อัตโนมัติ
ถ้า build ไม่ผ่าน update จะไม่หยุดบอทเดิม
ถ้า systemd รายงานว่าเริ่มโปรแกรมไม่สำเร็จ update จะคืน binary/service เดิมและพยายาม restart
การคืน binary ไม่ย้อน schema หรือข้อมูล SQLite และไม่รับประกันตรวจพบปัญหาที่เกิดหลังเริ่มสำเร็จ
การเปลี่ยน schema ในฟีเจอร์ใหม่ต้องมี migration/backup ที่เหมาะสมต่างหาก
