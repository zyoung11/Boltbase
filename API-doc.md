### Boltbase 前端 - API 文档

---

#### **1. 获取主页**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/`

**URL参数**：无

**表单参数**：无

**返回**：`web/views/index.html`

---

#### **2. 获取favicon**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/favicon.ico`

**URL参数**：无

**表单参数**：无

**返回**：`web/public/favicon.ico`

---

#### **3. 获取所有桶**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/getBuckets`

**URL参数**：无

**表单参数**：无

**返回**：
 - `BucketList` （切片类型）
 - `web/views/HTMX/getBucket.html`

---

#### **4. 获取一个桶的所有数据** （待弃用）
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/getBuckets`

**URL参数**：无

**表单参数**：`bucketName`（string类型）

**返回**：
 - `kv`（字典类型）
 - `Count` （int类型）
 - `web/views/HTMX/getAll.html`

---

#### **5. 选择桶**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/setBucket/{bucketName}`

**URL参数**：`bucketName`（string类型）

**表单参数**：无

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `web/views/HTMX/getPart.html`

---

#### **6. 选择页**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/setPage/{page}`

**URL参数**：`page`（int类型）

**表单参数**：无

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `web/views/HTMX/getPart.html`

---

#### **7. 选择一页键值对数量**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/setStep/{step}`

**URL参数**：`step`（int类型）

**表单参数**：无

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `web/views/HTMX/getPart.html`

---

#### **8. 上一页&下一页**
**HTTP方法**：GET 

**URL**：`http://localhost:5090/web/changePage/{direction}`

**URL参数**：`direction`（string类型）

**表单参数**：无

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `web/views/HTMX/getPart.html`

---
