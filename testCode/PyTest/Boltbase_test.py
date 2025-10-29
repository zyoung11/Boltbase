from APITEST import run_test, post, get, put, delete

baseUrl = "http://localhost:5090"

run_test("创建string类型的桶", post(f"{baseUrl}/bucket/test-string/string"))
run_test("创建seq类型的桶", post(f"{baseUrl}/bucket/test-seq/seq"))
run_test("创建time类型的桶", post(f"{baseUrl}/bucket/test-time/time"))
run_test("创建test桶", post(f"{baseUrl}/bucket/test/seq"))

run_test("查看所有的桶", get(f"{baseUrl}/bucket"))

run_test("修改test桶的桶名", put(f"{baseUrl}/bucket/test/test-new"))

run_test("查看所有的桶", get(f"{baseUrl}/bucket"))

run_test("获取桶的类型", get(f"{baseUrl}/bucket/type"))

run_test("删除桶", delete(f"{baseUrl}/bucket/test-new"))

run_test("查看所有的桶", get(f"{baseUrl}/bucket"))

run_test("向string类型的桶插入数据_1",
post(f"{baseUrl}/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-1",
             "Update": true
           }'''))

run_test("读取test-string桶的所以数据_1", get(f"{baseUrl}/kv/all/test-string"))

run_test("向string类型的桶插入数据_2",
post(f"{baseUrl}/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-2",
             "Update": true
           }'''))

run_test("读取test-string桶的所以数据_2", get(f"{baseUrl}/kv/all/test-string"))

run_test("向string类型的桶插入冲突数据_3",
post(f"{baseUrl}/kv",
  should_fail=True,
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-3",
             "Update": false
           }'''))

run_test("读取test-string表的所以数据_3", get(f"{baseUrl}/kv/all/test-string"))

seq_1 = run_test("向seq类型的桶插入数据",
post(f"{baseUrl}/kv",
  extract="key",
  body='''{
             "Bucket": "test-seq",
             "Value": "test-value"
           }'''))

run_test("读取test-seq表的所以数据", get(f"{baseUrl}/kv/all/test-seq"))

time_1 = run_test("向time类型的桶插入数据",
post(f"{baseUrl}/kv",
  extract="key",
  body='''{
             "Bucket": "test-time",
             "Value": "test-value"
           }'''))

run_test("读取test-time表的所以数据", get(f"{baseUrl}/kv/all/test-time"))


run_test("向string类型的桶插入数据",
post(f"{baseUrl}/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key-test",
             "Value": "test-value-test",
             "Update": true
           }'''))

seqKey = run_test("向seq类型的桶插入数据",
post(f"{baseUrl}/kv",
  body='''{
             "Bucket": "test-seq",
             "Value": "test-value-test"
           }''',
  extract="key"))

timeKey = run_test("向time类型的桶插入数据",
post(f"{baseUrl}/kv",
  body='''{
             "Bucket": "test-time",
             "Value": "test-value-test"
           }''',
  extract="key"))

run_test("读取test-string桶的所以数据", get(f"{baseUrl}/kv/all/test-string"))
run_test("读取test-seq桶的所以数据", get(f"{baseUrl}/kv/all/test-seq"))
run_test("读取test-time桶的所以数据", get(f"{baseUrl}/kv/all/test-time"))

run_test("删除test-string桶的一个数据", delete(f"{baseUrl}/kv/test-string/test-key-test"))
run_test("删除test-seq桶的一个数据", delete(f"{baseUrl}/kv/test-seq/{seqKey}"))
run_test("删除test-time桶的一个数据", delete(f"{baseUrl}/kv/test-time/{timeKey}"))

run_test("读取test-string桶的所以数据", get(f"{baseUrl}/kv/all/test-string"))
run_test("读取test-seq桶的所以数据", get(f"{baseUrl}/kv/all/test-seq"))
run_test("读取test-time桶的所以数据", get(f"{baseUrl}/kv/all/test-time"))

run_test("查询test-string桶的一个数据", get(f"{baseUrl}/kv/get/test-string/test-key"))
run_test("查询test-seq桶的一个数据", get(f"{baseUrl}/kv/get/test-seq/{seq_1}"))
run_test("查询test-time桶的一个数据", get(f"{baseUrl}/kv/get/test-time/{time_1}"))
