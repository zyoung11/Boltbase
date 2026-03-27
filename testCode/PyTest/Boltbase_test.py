from PAT import run_test, post, get, put, delete, print_info, show_result
import base64

baseUrl = "http://localhost:5090"

# ==================== 认证系统测试 ====================

# 1. 健康检查（无需认证）
run_test("健康检查", get(f"{baseUrl}/health", should_fail=True))

# 2. 无密码模式下创建管理员密码（首次创建无需认证）
admin_user = "admin"
admin_pass = "securepassword123"
run_test(
    "创建管理员密码",
    post(
        f"{baseUrl}/auth/password",
        body={
            "Username": admin_user,
            "Password": admin_pass
        }
    )
)

# 3. 使用错误密码测试认证失败（预期失败）
wrong_credentials = base64.b64encode(f"{admin_user}:wrongpassword".encode()).decode()
run_test(
    "使用错误密码访问桶列表（预期失败）",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": f"Basic {wrong_credentials}"},
        should_fail=True
    )
)

# 4. 使用正确密码测试认证成功
correct_credentials = base64.b64encode(f"{admin_user}:{admin_pass}".encode()).decode()
run_test(
    "使用正确密码访问桶列表",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 5. 创建API密钥
apikey_response = run_test(
    "创建API密钥（有效期24小时）",
    post(
        f"{baseUrl}/auth/apikey",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Duration": "24h"
        }
    ),
    "apiKey"
)

# 6. 使用API密钥访问桶列表
run_test(
    "使用API密钥访问桶列表",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": apikey_response}
    )
)

# 7. 测试API密钥不能访问管理员专属端点（导出数据库）
run_test(
    "API密钥尝试导出数据库（预期失败）",
    post(
        f"{baseUrl}/export",
        headers={"Authorization": apikey_response},
        should_fail=True
    )
)

# 8. 清理过期的API密钥
run_test(
    "清理过期的API密钥",
    delete(
        f"{baseUrl}/auth/apikey",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# ==================== Bucket管理测试 ====================

# 9. 创建不同类型的桶
run_test(
    "创建string类型的桶",
    post(
        f"{baseUrl}/bucket/test-string/string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "创建seq类型的桶",
    post(
        f"{baseUrl}/bucket/test-seq/seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "创建time类型的桶",
    post(
        f"{baseUrl}/bucket/test-time/time",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "创建test桶",
    post(
        f"{baseUrl}/bucket/test/seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 10. 列出所有桶
run_test(
    "查看所有的桶",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 11. 重命名桶
run_test(
    "修改test桶的桶名",
    put(
        f"{baseUrl}/bucket/test/test-new",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 12. 再次列出所有桶
run_test(
    "查看所有的桶（重命名后）",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 13. 获取桶的类型
run_test(
    "获取桶的类型",
    get(
        f"{baseUrl}/bucket/type",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 14. 删除桶
run_test(
    "删除桶",
    delete(
        f"{baseUrl}/bucket/test-new",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 15. 最终查看所有桶
run_test(
    "查看所有的桶（删除后）",
    get(
        f"{baseUrl}/bucket",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# ==================== Key-Value操作测试 ====================

# 16. 向string类型的桶插入数据（测试更新）
run_test(
    "向string类型的桶插入数据_1",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "test-key",
            "Value": "test-value-1",
            "Update": True
        }
    )
)

# 17. 读取string桶的所有数据
run_test(
    "读取test-string桶的所有数据_1",
    get(
        f"{baseUrl}/kv/all/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 18. 更新string桶的数据
run_test(
    "向string类型的桶插入数据_2（更新）",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "test-key",
            "Value": "test-value-2",
            "Update": True
        }
    )
)

# 19. 再次读取string桶的数据
run_test(
    "读取test-string桶的所有数据_2（更新后）",
    get(
        f"{baseUrl}/kv/all/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 20. 测试string桶的插入冲突（Update=false时key已存在）
run_test(
    "向string类型的桶插入冲突数据_3（预期失败）",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "test-key",
            "Value": "test-value-3",
            "Update": False
        },
        should_fail=True
    )
)

# 21. 读取string桶的数据（验证冲突后数据未变）
run_test(
    "读取test-string表的所有数据_3（冲突后）",
    get(
        f"{baseUrl}/kv/all/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 22. 向seq类型的桶插入数据（自动生成key）
seq_response_1 = run_test(
    "向seq类型的桶插入数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-seq",
            "Value": "test-value"
        }
    ),
    "key"
)

# 23. 读取seq桶的所有数据
run_test(
    "读取test-seq表的所有数据",
    get(
        f"{baseUrl}/kv/all/test-seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 24. 向time类型的桶插入数据（自动生成key）
time_response_1 = run_test(
    "向time类型的桶插入数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-time",
            "Value": "test-value"
        }
    ),
    "key"
)

# 25. 读取time桶的所有数据
run_test(
    "读取test-time表的所有数据",
    get(
        f"{baseUrl}/kv/all/test-time",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 26. 向string桶插入新数据
run_test(
    "向string类型的桶插入数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "test-key-test",
            "Value": "test-value-test",
            "Update": True
        }
    )
)

# 27. 向seq桶插入新数据
seqKey = run_test(
    "向seq类型的桶插入数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-seq",
            "Value": "test-value-test"
        }
    ),
    "key"
)

# 28. 向time桶插入新数据
timeKey = run_test(
    "向time类型的桶插入数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-time",
            "Value": "test-value-test"
        }
    ),
    "key"
)

# 29. 读取所有桶的数据
run_test(
    "读取test-string桶的所有数据",
    get(
        f"{baseUrl}/kv/all/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "读取test-seq桶的所有数据",
    get(
        f"{baseUrl}/kv/all/test-seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "读取test-time桶的所有数据",
    get(
        f"{baseUrl}/kv/all/test-time",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 30. 删除键值对
run_test(
    "删除test-string桶的一个数据",
    delete(
        f"{baseUrl}/kv/test-string/test-key-test",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "删除test-seq桶的一个数据",
    delete(
        f"{baseUrl}/kv/test-seq/{seqKey}",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "删除test-time桶的一个数据",
    delete(
        f"{baseUrl}/kv/test-time/{timeKey}",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# ==================== 数据查询测试 ====================

# 31. 查询所有数据（删除后）
run_test(
    "查询test-string桶的全部数据",
    get(
        f"{baseUrl}/kv/all/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "查询test-seq桶的全部数据",
    get(
        f"{baseUrl}/kv/all/test-seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "查询test-time桶的全部数据",
    get(
        f"{baseUrl}/kv/all/test-time",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 32. 查询单个键值对
run_test(
    "查询test-string桶的一个数据",
    get(
        f"{baseUrl}/kv/get/test-string/test-key",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "查询test-seq桶的一个数据",
    get(
        f"{baseUrl}/kv/get/test-seq/{seq_response_1}",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "查询test-time桶的一个数据",
    get(
        f"{baseUrl}/kv/get/test-time/{time_response_1}",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 33. 前缀扫描测试（需要先添加一些有前缀的数据）
run_test(
    "添加带前缀的数据1",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "user:100",
            "Value": "Alice",
            "Update": True
        }
    )
)

run_test(
    "添加带前缀的数据2",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "user:101",
            "Value": "Bob",
            "Update": True
        }
    )
)

run_test(
    "添加不匹配前缀的数据",
    post(
        f"{baseUrl}/kv",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Bucket": "test-string",
            "Key": "product:1",
            "Value": "Laptop",
            "Update": True
        }
    )
)

run_test(
    "前缀扫描测试",
    get(
        f"{baseUrl}/kv/prefix/test-string/user:",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 34. 范围扫描测试（seq类型）
# 先添加一些seq数据
for i in range(2, 6):
    run_test(
        f"添加seq数据{i}",
        post(
            f"{baseUrl}/kv",
            headers={"Authorization": f"Basic {correct_credentials}"},
            body={
                "Bucket": "test-seq",
                "Value": f"seq-value-{i}"
            }
        )
    )

run_test(
    "范围扫描测试（seq类型）",
    get(
        f"{baseUrl}/kv/range/test-seq/1/4",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# ==================== 信息与导出测试 ====================

# 35. 统计键值对数量
run_test(
    "统计test-string桶的键值对数量",
    get(
        f"{baseUrl}/kv/count/test-string",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

run_test(
    "统计test-seq桶的键值对数量",
    get(
        f"{baseUrl}/kv/count/test-seq",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 36. 导出数据库（仅限管理员）
run_test(
    "导出数据库",
    post(
        f"{baseUrl}/export",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# ==================== 清理测试 ====================

# 37. 测试删除管理员密码前需要删除API密钥桶的限制
# 首先创建一个API密钥（此时会创建API密钥桶）
run_test(
    "创建新的API密钥（创建API密钥桶）",
    post(
        f"{baseUrl}/auth/apikey",
        headers={"Authorization": f"Basic {correct_credentials}"},
        body={
            "Duration": "1h"
        }
    )
)

# 尝试删除管理员密码（应该失败，因为存在API密钥桶）
run_test(
    "尝试删除管理员密码（存在API密钥桶，预期失败）",
    delete(
        f"{baseUrl}/auth/password",
        headers={"Authorization": f"Basic {correct_credentials}"},
        should_fail=True
    )
)

# 38. 删除API密钥桶
run_test(
    "删除API密钥桶",
    delete(
        f"{baseUrl}/bucket/BoltbaseApiKeyBucket",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 39. 现在可以删除管理员密码
run_test(
    "删除管理员密码",
    delete(
        f"{baseUrl}/auth/password",
        headers={"Authorization": f"Basic {correct_credentials}"}
    )
)

# 40. 验证系统回到无密码模式
run_test(
    "验证系统回到无密码模式",
    get(f"{baseUrl}/bucket")
)

# ==================== 测试结果汇总 ====================
print_info(
    "测试信息",
    {
        "测试项目": "Boltbase API 端到端测试",
        "测试框架": "PAT",
        "测试范围": "所有API端点",
        "认证模式": "无密码模式 -> 管理员模式 -> API密钥模式 -> 清理"
    }
)

show_result("Boltbase API测试结果汇总", True)
