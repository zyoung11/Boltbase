## Boltbase Web - API 文档

---

### **1. 获取主页**

**HTTP方法**：`GET `

**URL**：`http://localhost:5090/`

**返回**：`web/views/index.html`

---

### **2. 获取favicon**

**HTTP方法**：`GET `

**URL**：`http://localhost:5090/favicon.ico`

**返回**：`web/public/favicon.ico`

---

### **3. 获取所有桶**

**HTTP方法**：`GET`

**URL**：`http://localhost:5090/web/getBuckets`

**返回**：
 - `BucketList` （切片类型）
 - `web/views/HTMX/getBucket.html`

---

### **4. 选择桶**

**HTTP方法**：`GET` 

**URL**：`http://localhost:5090/web/setBucket/{bucketName}`

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `totalKV` (int类型：返回桶内一共有多少键值对)
 - `totalPage`（int类型： 返回一共有多少页）
 - `currentPage`（int类型： 返回当前在哪页）
 - `bucketName`（string类型：当前在使用的桶）
 - `pageButtons`（切片类型：一共9个string类型的元素，从前到后对应9个按钮）
 - `web/views/HTMX/getPart.html`

---

### **5. 选择页**

**HTTP方法**：`GET` 

**URL**：`http://localhost:5090/web/setPage/{page}`

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `totalKV` (int类型：返回桶内一共有多少键值对)
 - `totalPage`（int类型： 返回一共有多少页）
 - `currentPage`（int类型： 返回当前在哪页）
 - `bucketName`（string类型：当前在使用的桶）
 - `pageButtons`（切片类型：一共9个string类型的元素，从前到后对应9个按钮）
 - `web/views/HTMX/getPart.html`

---

### **6. 选择一页键值对数量**

**HTTP方法**：`GET` 

**URL**：`http://localhost:5090/web/setStep/{step}`

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `totalKV` (int类型：返回桶内一共有多少键值对)
 - `totalPage`（int类型： 返回一共有多少页）
 - `currentPage`（int类型： 返回当前在哪页）
 - `bucketName`（string类型：当前在使用的桶）
 - `pageButtons`（切片类型：一共9个string类型的元素，从前到后对应9个按钮）
 - `web/views/HTMX/getPart.html`

---

### **7. 上一页&下一页**

**HTTP方法**：`GET` 

**URL**：`http://localhost:5090/web/changePage/{direction}`

**返回**：
 - `kv`（字典类型）
 - `total` （int类型：返回的键值对数量）
 - `totalKV` (int类型：返回桶内一共有多少键值对)
 - `totalPage`（int类型： 返回一共有多少页）
 - `currentPage`（int类型： 返回当前在哪页）
 - `bucketName`（string类型：当前在使用的桶）
 - `pageButtons`（切片类型：一共9个string类型的元素，从前到后对应9个按钮）
 - `web/views/HTMX/getPart.html`

---

### **8. 获取桶元数据**

**HTTP方法**：`GET` 

**URL**：`http://localhost:5090/web/info/{bucketName}`

**返回**：
 - `Info`（字典类型：键是元数据的名字，string类型；值是对应的数据，int类型）

---
