from APITEST import run_test, post, get, put, delete

run_test("创建string类型的桶", post("http://localhost:5090/bucket/test-string/string"))
run_test("创建seq类型的桶", post("http://localhost:5090/bucket/test-seq/seq"))
run_test("创建time类型的桶", post("http://localhost:5090/bucket/test-time/time"))
run_test("创建test桶", post("http://localhost:5090/bucket/test/seq"))

run_test("查看所有的桶", get("http://localhost:5090/bucket"))

run_test("修改test桶的桶名", put("http://localhost:5090/bucket/test/test-new"))

run_test("查看所有的桶", get("http://localhost:5090/bucket"))

run_test("获取桶的类型", get("http://localhost:5090/bucket/type"))

run_test("删除桶", delete("http://localhost:5090/bucket/test-new"))

run_test("查看所有的桶", get("http://localhost:5090/bucket"))

run_test("向string类型的桶插入数据_1",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-1",
             "Update": true
           }'''))

run_test("读取test-string桶的所以数据_1", get("http://localhost:5090/kv/all/test-string"))

run_test("向string类型的桶插入数据_2",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-2",
             "Update": true
           }'''))

run_test("读取test-string桶的所以数据_2", get("http://localhost:5090/kv/all/test-string"))

run_test("向string类型的桶插入冲突数据_3",
post("http://localhost:5090/kv",
  should_fail=True,
  body='''{
             "Bucket": "test-string",
             "Key": "test-key",
             "Value": "test-value-3",
             "Update": false
           }'''))

run_test("读取test-string表的所以数据_3", get("http://localhost:5090/kv/all/test-string"))

run_test("向seq类型的桶插入数据",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-seq",
             "Value": "test-value"
           }'''))

run_test("读取test-seq表的所以数据", get("http://localhost:5090/kv/all/test-seq"))

run_test("向time类型的桶插入数据",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-time",
             "Value": "test-value"
           }'''))

run_test("读取test-time表的所以数据", get("http://localhost:5090/kv/all/test-time"))


run_test("向string类型的桶插入数据",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-string",
             "Key": "test-key-test",
             "Value": "test-value-test",
             "Update": true
           }'''))

seqKey = run_test("向seq类型的桶插入数据",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-seq",
             "Value": "test-value-test"
           }''',
  extract="key"))

timeKey = run_test("向time类型的桶插入数据",
post("http://localhost:5090/kv",
  body='''{
             "Bucket": "test-time",
             "Value": "test-value-test"
           }''',
  extract="key"))

run_test("读取test-string表的所以数据", get("http://localhost:5090/kv/all/test-string"))
run_test("读取test-seq表的所以数据", get("http://localhost:5090/kv/all/test-seq"))
run_test("读取test-time表的所以数据", get("http://localhost:5090/kv/all/test-time"))

run_test("删除test-string表的一个数据", delete("http://localhost:5090/kv/test-string/test-key-test"))
run_test("删除test-seq表的一个数据", delete(f"http://localhost:5090/kv/test-seq/{seqKey}"))
run_test("删除test-time表的一个数据", delete(f"http://localhost:5090/kv/test-time/{timeKey}"))

run_test("读取test-string表的所以数据", get("http://localhost:5090/kv/all/test-string"))
run_test("读取test-seq表的所以数据", get("http://localhost:5090/kv/all/test-seq"))
run_test("读取test-time表的所以数据", get("http://localhost:5090/kv/all/test-time"))

