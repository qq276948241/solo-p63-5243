# 社区诊所预约挂号后端

Go + Gin 实现的社区诊所预约挂号系统后端。支持患者查排班、预约/取消预约、就诊后评价；医生查看当日预约、标记就诊完成、临时停诊。

## 技术栈

- **框架**：Gin
- **ORM**：GORM v1
- **数据库**：MySQL 8.x
- **认证**：JWT
- **密码加密**：bcrypt

## 项目结构

```
project63/
├── cmd/
│   └── server/
│       └── main.go          # 主程序入口
├── internal/
│   ├── common/               # 公共模块（配置、数据库、JWT、中间件、响应）
│   ├── user/                 # 用户模块（注册/登录/医生列表）
│   │   ├── model.go
│   │   ├── service.go
│   │   └── handler.go
│   ├── schedule/             # 排班模块（查排班/创建排班/停诊）
│   │   ├── model.go
│   │   ├── service.go
│   │   └── handler.go
│   └── appointment/          # 预约模块（预约/取消/完成/评价）
│       ├── model.go
│       ├── service.go
│       └── handler.go
├── scripts/
│   └── seed/
│       └── main.go           # 数据初始化脚本
├── .env.example              # 环境变量示例
└── go.mod
```

每个模块严格按照 **handler → service → model** 三层组织：

- **handler**：处理 HTTP 路由、参数绑定、权限校验，返回统一响应
- **service**：核心业务逻辑、事务控制、数据一致性保障
- **model**：数据库表结构定义、DTO、领域方法（状态校验、归属判断）

## 模块说明

### 1. user 模块 — 用户与认证

负责用户注册登录、JWT 签发、医生资料维护。密码统一使用 bcrypt 加密存储。

**角色**：
- `patient` — 患者：可查排班、预约、取消预约、评价
- `doctor` — 医生：可创建排班、停诊、查看当日预约、标记就诊完成

### 2. schedule 模块 — 医生排班

管理医生的出诊排班。每个排班包含日期、时间段、最大接诊数、已预约数、状态。

**排班状态**：
- `available` — 可预约
- `canceled` — 已停诊

### 3. appointment 模块 — 预约与评价

处理预约全生命周期和评价功能。所有写操作使用 **事务 + `SELECT ... FOR UPDATE` 行锁** 保证并发安全，状态变更严格遵循状态机校验。

**预约状态**：
- `pending` — 待确认
- `confirmed` — 已确认
- `completed` — 就诊完成（可评价）
- `canceled` — 已取消

**状态流转**：
```
patient 预约 → confirmed
patient 取消 → canceled  （仅 pending/confirmed）
doctor  完成 → completed （仅 pending/confirmed）
patient 评价              （仅 completed，且每个预约仅能评价一次）
```

## 数据库表

| 表名 | 说明 |
|------|------|
| `users` | 用户表（患者 + 医生共用） |
| `doctor_profiles` | 医生资料表（科室、职称、简介） |
| `schedules` | 排班表 |
| `appointments` | 预约表（关联医生、患者、排班） |
| `reviews` | 评价表（关联预约、医生、患者，1-5 星评分 + 评语） |

服务启动时自动执行 GORM AutoMigrate。

## API 接口

所有接口统一返回格式：
```json
{ "code": 0, "message": "success", "data": {} }
```
`code=0` 表示成功，非 0 表示失败。

需要认证的接口请求头携带：`Authorization: Bearer <token>`

---

### user 模块（`/api/user`）

#### 注册
```
POST /api/user/register
```
```json
{
  "username": "zhangsan",
  "password": "123456",
  "real_name": "张三",
  "role": "patient",
  "phone": "13800138000",
  "gender": "男",
  "age": 30
}
```
- `role`: `patient` 或 `doctor`

#### 登录
```
POST /api/user/login
```
```json
{
  "username": "zhangsan",
  "password": "123456"
}
```
响应：
```json
{
  "code": 0,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIs...",
    "user": { /* 用户信息 */ }
  }
}
```

#### 获取当前用户信息
```
GET /api/user/me
Authorization: Bearer <token>
```

#### 获取医生列表
```
GET /api/user/doctors
Authorization: Bearer <token>
```

---

### schedule 模块（`/api/schedule`，全部需登录）

#### 查询排班列表
```
GET /api/schedule?doctor_id=1&date=2026-06-24
GET /api/schedule?start_date=2026-06-24&end_date=2026-06-30
```
查询参数：
- `doctor_id` — 按医生过滤
- `date` — 按具体日期过滤
- `start_date` / `end_date` — 按日期范围过滤

#### 获取排班详情
```
GET /api/schedule/:id
```

#### 创建排班（仅医生）
```
POST /api/schedule
Authorization: Bearer <doctor-token>
```
```json
{
  "doctor_id": 1,
  "date": "2026-06-25",
  "start_time": "09:00",
  "end_time": "09:30",
  "max_count": 1
}
```

#### 我的当日排班（仅医生）
```
GET /api/schedule/doctor/today?date=2026-06-24
Authorization: Bearer <doctor-token>
```
`date` 不传默认今天。

#### 临时停诊（仅医生）
```
POST /api/schedule/:id/cancel
Authorization: Bearer <doctor-token>
```

---

### appointment 模块（`/api/appointment`，全部需登录）

#### 创建预约（仅患者）
```
POST /api/appointment
Authorization: Bearer <patient-token>
```
```json
{
  "schedule_id": 1,
  "symptoms": "头晕、血压高"
}
```
- 自动做时段冲突校验：同一医生同一时段不能被两个患者预约，同一患者同一时段不能重复预约
- 自动扣减排班已预约数

#### 我的预约列表
```
GET /api/appointment/my?status=confirmed&date=2026-06-24
Authorization: Bearer <token>
```
患者视角返回自己的预约，医生视角返回自己的预约。

#### 获取预约详情
```
GET /api/appointment/:id
Authorization: Bearer <token>
```
仅预约关联的患者或医生可见。

#### 取消预约（仅患者）
```
POST /api/appointment/:id/cancel
Authorization: Bearer <patient-token>
```
仅 `pending` / `confirmed` 状态可取消，自动回滚排班已预约数。

#### 标记就诊完成（仅医生）
```
POST /api/appointment/:id/complete
Authorization: Bearer <doctor-token>
```
```json
{
  "remark": "血压偏高，已开药，建议一周后复诊"
}
```
仅 `pending` / `confirmed` 状态可标记完成。

#### 当日预约列表（仅医生）
```
GET /api/appointment/doctor/today?date=2026-06-24
Authorization: Bearer <doctor-token>
```

#### 患者评价（仅患者）
```
POST /api/appointment/:id/review
Authorization: Bearer <patient-token>
```
```json
{
  "rating": 5,
  "comment": "医生很专业，讲解很详细，态度也很好"
}
```
- 仅 `completed` 状态的预约可评价
- 每个预约只能评价一次
- `rating` 范围 1-5

#### 查看医生评分
```
GET /api/appointment/doctor/:id/rating
Authorization: Bearer <token>
```
响应：
```json
{
  "code": 0,
  "data": {
    "doctor_id": 1,
    "avg_rating": 4.8,
    "total_count": 25,
    "rating_count": {
      "1": 0, "2": 1, "3": 2, "4": 5, "5": 17
    }
  }
}
```

#### 查看医生评价列表
```
GET /api/appointment/doctor/:id/reviews
Authorization: Bearer <token>
```

---

## 本地运行

### 1. 准备数据库

MySQL 中创建数据库：
```sql
CREATE DATABASE clinic DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 配置环境变量

```bash
cp .env.example .env
```

按需修改 `.env`：
```
DB_HOST=127.0.0.1
DB_PORT=3306
DB_USER=root
DB_PASSWORD=root
DB_NAME=clinic

JWT_SECRET=clinic-secret-key-change-in-production
JWT_EXPIRE=86400

SERVER_PORT=8080
```

### 3. 编译并启动服务

```bash
# 编译
go build -o bin/server.exe ./cmd/server

# 启动
./bin/server.exe
# 或直接运行
go run ./cmd/server/main.go
```

服务启动后会自动执行数据库表迁移。访问 `http://localhost:8080/health` 检查健康状态。

## Seed 数据初始化

脚本会向数据库插入 **4 名医生 + 1 名测试患者 + 未来 7 天的排班数据**，方便快速联调。

```bash
# 编译
go build -o bin/seed.exe ./scripts/seed

# 执行
./bin/seed.exe
# 或直接运行
go run ./scripts/seed/main.go
```

执行完成后终端会输出测试账号：

| 角色 | 用户名 | 密码 | 科室 |
|------|--------|------|------|
| 医生 | zhangyisheng | 123456 | 内科 主任医师 |
| 医生 | liyisheng | 123456 | 外科 副主任医师 |
| 医生 | wangyisheng | 123456 | 儿科 主治医师 |
| 医生 | zhaoyisheng | 123456 | 妇产科 副主任医师 |
| 患者 | patient1 | 123456 | — |

排班时段：每天 08:00-11:00、14:00-17:00，每 30 分钟一个时段，每个时段最大接诊 1 人。

## 健康检查

```
GET /health
```
响应：`{ "status": "ok" }`
