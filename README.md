# Boltbase

Boltbase 是一个基于 Go 语言和 [bbolt](https://github.com/etcd-io/bbolt) 数据库构建的、可通过 RESTful API 进行交互的高性能键值(Key-Value)存储服务。它通过 [Fiber](https://github.com/gofiber/fiber) 框架提供 Web 服务，设计上兼顾了开发的便捷性与生产环境的安全性。

## ✨ 功能特性

- **嵌入式键值数据库**: 基于 `bbolt`，提供持久化的本地数据存储，无需额外数据库服务。
- **RESTful API**: 提供清晰、简单的 HTTP 接口用于数据操作。
- **多种主键策略**: 支持自定义字符串(string)、自增序列(seq)和时间序列(time)作为主键类型。
- **灵活的认证系统**:
    - **无密码开发模式**: 方便快速启动和开发测试。
    - **管理员密码模式**: 通过 Basic Auth 提供基础的管理员认证。
    - **API 密钥模式**: 为应用或服务提供安全的、可过期的访问令牌。
- **强大的查询功能**: 支持单键获取、前缀扫描、范围扫描和全量扫描。
- **动态存储桶 (Bucket) 管理**: 可通过 API 创建、重命名、删除和列出 Buckets。
- **健康检查**: 内置 `/health` 端点，方便集成到容器编排或服务监控系统中。
- **数据导出**: 支持将整个数据库导出为 JSON 格式，便于备份和迁移。

## 🚀 如何开始

### 1. 环境准备
确保您已经安装了 Go (建议版本 1.18+)。

### 2. 下载与运行
```bash
# 克隆项目 (或者直接下载代码)
# git clone https://github.com/zyoung11/Boltbase.git
# cd boltbase

# 安装依赖
go mod tidy

# 运行服务
go run .
```
服务默认启动在 `5090` 端口。

### 3. 配置文件
Boltbase 的数据将存储在运行目录下的 `Boltbase.db` 文件中。

## 🔑 认证系统

Boltbase 的认证系统设计得非常灵活，以适应不同场景的需求。所有需要认证的请求都通过 `Authorization` HTTP Header 传递凭证。

### 阶段一：无密码开发模式 (默认)

在数据库中**不存在**管理员账户时，Boltbase 处于无密码的开发模式。在此模式下：
- **所有 API 端点都开放访问**，无需提供 `Authorization` 头。
- 这是项目的初始状态，便于开发者快速进行功能测试和集成。
- 您可以通过调用 `POST /auth/password` 来创建一个管理员账户，从而切换到安全模式。

### 阶段二：管理员密码模式

当您通过 `POST /auth/password` 成功创建管理员后，系统会进入此模式。
- **认证方式**: HTTP Basic Authentication。
- **凭证格式**: `Authorization: Basic <base64_encoded_username:password>`
- 在此模式下，除了 `/health` 端点，所有 API 请求都**必须**提供正确的管理员凭证。
- 拥有管理员权限后，您可以开始创建 API 密钥供其他应用使用。

### 阶段三：API 密钥模式

在管理员模式下，您可以通过 `POST /auth/apikey` 创建有时效性的 API 密钥。
- **认证方式**: 直接在 `Authorization` Header 中传递密钥。
- **凭证格式**: `Authorization: <your_api_key>`
- API 密钥用于非管理员权限的普通操作，它无法访问管理员专属的端点（如用户管理、导出数据库等）。
- 每个密钥都有一个过期时间，过期的密钥将无法通过认证。

**逻辑总结**:
1.  系统初始化为无密码模式。
2.  创建管理员密码后，进入管理员模式，所有操作需要管理员凭证。
3.  管理员可以创建 API 密钥，持有密钥者可进行普通数据操作。
4.  如果删除了管理员密码（且没有设置任何 API 密钥），系统将退回至无密码开发模式。

---

## 📚 API 文档

以下是 Boltbase 提供的所有 API 端点的详细说明。

### 一、健康检查

#### **1.1** `GET /health`
检查服务是否正在运行。
- **认证**: 无
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**: `OK`

---
### 二、认证管理

#### **2.1** `POST /auth/password`
创建或更新管理员密码。首次创建后，系统将进入“管理员密码模式”。
- **认证**:
    - 首次创建: 无
    - 更新密码: 需要有效的管理员 `Authorization` Header。
- **请求体** (`application/json`):
  ```json
  {
    "Username": "admin",
    "Password": "your_strong_password"
  }
  ```
- **成功响应**:
    - **Code**: `201 Created`
---
#### **2.2** `DELETE /auth/password`
删除管理员密码。如果当前没有设置 API 密钥，系统将回到“无密码开发模式”。
- **认证**: 需要有效的管理员 `Authorization` Header。
- **限制**: 如果数据库中存在 API 密钥桶 (`BoltbaseApiKeyBucket`)，则无法删除密码，必须先删除 API 密钥桶。
- **成功响应**:
    - **Code**: `200 OK`
---
#### **2.3** `POST /auth/apikey`
创建一个有时效性的 API 密钥。
- **认证**: 需要有效的管理员 `Authorization` Header。
- **请求体** (`application/json`):
  ```json
  {
    "Duration": "24h" 
  }
  ```
- **Duration 有效单位**:
  `Duration` 字符串可以组合使用以下单位，例如 `"1w2d6h"` 表示 1 周 2 天 6 小时。

| 单位 | 描述 |
| :--- | :--- |
| `w` | 周 (Weeks) |
| `d` | 天 (Days) |
| `h` | 小时 (Hours) |
| `m` | 分钟 (Minutes) |
| `s` | 秒 (Seconds) |
| `ms` | 毫秒 (Milliseconds) |
| `us` or `µs` | 微秒 (Microseconds) |
| `ns` | 纳秒 (Nanoseconds) |

- **成功响应**:
    - **Code**: `201 Created`
    - **Body**:
      ```json
      {
        "apiKey": "a1b2c3d4-e5f6-7890-1234-567890abcdef",
        "expiryTime": "2025-08-16T12:00:00Z"
      }
      ```
---
#### **2.4** `DELETE /auth/apikey`
清理所有已过期的 API 密钥。
- **认证**: 需要有效的管理员 `Authorization` Header。
- **成功响应**:
    - **Code**: `204 No Content`

---
### 三、Bucket (存储桶) 管理

#### **3.1** `POST /bucket/:bucketName/:keyType`
创建一个新的 Bucket。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 的名称。
    - `keyType` (string, required): Bucket 的主键类型。可选值: `string`, `seq`, `time`。
- **成功响应**:
    - **Code**: `201 Created`
---
#### **3.2** `GET /bucket`
列出所有可访问的 Bucket。
- **认证**: 需要
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "BucketList": ["users", "products", "logs"],
        "total": 3
      }
      ```
---
#### **3.3** `GET /bucket/type`
列出所有 Bucket 及其主键类型。
- **认证**: 需要
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "bucketTypeList": {
          "users": "string",
          "products": "seq",
          "logs": "time"
        }
      }
      ```
---
#### **3.4** `PUT /bucket/:oldName/:newName`
重命名一个 Bucket。
- **认证**: 需要
- **URL 参数**:
    - `oldName` (string, required): 旧的 Bucket 名称。
    - `newName` (string, required): 新的 Bucket 名称。
- **成功响应**:
    - **Code**: `204 No Content`
---
#### **3.5** `DELETE /bucket/:bucketName`
删除一个 Bucket 及其中的所有数据。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): 要删除的 Bucket 名称。
- **成功响应**:
    - **Code**: `204 No Content`

---
### 四、Key-Value (键值) 操作

#### **4.1** `POST /kv`
插入或更新一个键值对。
- **认证**: 需要
- **请求体** (`application/json`):
  ```json
  {
    "Bucket": "your_bucket_name",
    "Key": "your_key", // 在 keyType 为 'seq' 或 'time' 时可忽略
    "Value": "your_value",
    "Update": false // 仅在 keyType 为 'string' 时有效。true: 更新或插入; false: 仅当 key 不存在时插入
  }
  ```
- **行为说明**:
    - **`keyType: string`**: `Key` 字段为必填。
    - **`keyType: seq`**: `Key` 字段被忽略，自动生成自增 ID 作为键。
    - **`keyType: time`**: `Key` 字段被忽略，自动生成当前 UTC 时间作为键。
- **成功响应**:
    - **Code**: `201 Created`
---
#### **4.2** `DELETE /kv/:bucketName/:key`
删除一个键值对。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
    - `key` (string, required): 要删除的键。
- **成功响应**:
    - **Code**: `204 No Content`

---
### 五、数据查询

#### **5.1** `GET /kv/get/:bucketName/:key`
根据键获取一个值。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
    - `key` (string, required): 要查询的键。
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "value": "the_stored_value"
      }
      ```
---
#### **5.2** `GET /kv/prefix/:bucketName/:prefix`
在一个 Bucket 内进行前缀扫描，返回所有匹配前缀的键值对。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
    - `prefix` (string, required): 键的前缀。
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "total": 2,
        "kv": {
          "user:100": "Alice",
          "user:101": "Bob"
        }
      }
      ```
---
#### **5.3** `GET /kv/range/:bucketName/:start/:end`
在一个 Bucket 内进行范围扫描。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
    - `start` (string, required): 范围的起始键。
    - `end` (string, required): 范围的结束键。
- **注意**: 如果 Bucket 的 `keyType` 是 `seq`，`start` 和 `end` 应该是整数。
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "total": 5,
        "kv": {
          "key2": "value2",
          "key3": "value3",
          // ...
        }
      }
      ```
---
#### **5.4** `GET /kv/all/:bucketName`
获取一个 Bucket 中的所有键值对。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "total": 150,
        "kv": {
          "key1": "value1",
          "key2": "value2",
          // ...
        }
      }
      ```

---

### 六、信息与导出

#### **6.1** `GET /kv/count/:bucketName`
统计一个 Bucket 中的键值对总数。
- **认证**: 需要
- **URL 参数**:
    - `bucketName` (string, required): Bucket 名称。
- **成功响应**:
    - **Code**: `200 OK`
    - **Body**:
      ```json
      {
        "total": 150
      }
      ```
---
#### **6.2** `POST /export`
将整个数据库（所有 Buckets 和数据）导出为 `Boltbase.json` 文件。
- **认证**: **仅限管理员**
- **成功响应**:
    - **Code**: `201 Created`
    - **说明**: 文件将保存在 Boltbase 服务运行的目录下。