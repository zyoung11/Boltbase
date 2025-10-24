import requests
import json
from typing import Dict, Optional, Any, Tuple

def run_test(description: str, response: Tuple[str, Any]):
    status, content = response
    print(f"""
{description}: {status}
--------------------------------------------------
Response:
""", end="")
    try:
        # 尝试将内容作为JSON格式化输出
        print(json.dumps(content, indent=4, ensure_ascii=False))
    except (TypeError, json.JSONDecodeError):
        # 如果内容不是有效的JSON（例如，只是一个字符串），则直接打印
        print(content)
    print("--------------------------------------------------\n")

def post(url: str, body: Optional[str] = None, key: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    try:
        resp = requests.post(url, **kwargs)
        if 200 <= resp.status_code < 300:
            if should_fail:
                return ("❌", f"期望失败但成功: {resp.status_code}")
            else:
                try:
                    return ("✅", resp.json())
                except ValueError:
                    return ("✅", {"response": resp.text})
        else:
            if should_fail:
                return ("✅", {"error": f"状态码异常: {resp.status_code}"})
            else:
                return ("❌", f"状态码异常: {resp.status_code}")
    except Exception as e:
        if should_fail:
            return ("✅", str(e))
        else:
            return ("❌", str(e))

def delete(url: str, key: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.delete(url, headers=headers, timeout=10)
        if 200 <= resp.status_code < 300:
            if should_fail:
                return ("❌", f"期望失败但成功: {resp.status_code}")
            else:
                try:
                    return ("✅", resp.json())
                except ValueError:
                    return ("✅", {"response": resp.text})
        else:
            if should_fail:
                return ("✅", {"error": f"状态码异常: {resp.status_code}"})
            else:
                return ("❌", f"状态码异常: {resp.status_code}")
    except Exception as e:
        if should_fail:
            return ("✅", str(e))
        else:
            return ("❌", str(e))

def put(url: str, body: Optional[str] = None, key: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    try:
        resp = requests.put(url, **kwargs)
        if 200 <= resp.status_code < 300:
            if should_fail:
                return ("❌", f"期望失败但成功: {resp.status_code}")
            else:
                try:
                    return ("✅", resp.json())
                except ValueError:
                    return ("✅", {"response": resp.text})
        else:
            if should_fail:
                return ("✅", {"error": f"状态码异常: {resp.status_code}"})
            else:
                return ("❌", f"状态码异常: {resp.status_code}")
    except Exception as e:
        if should_fail:
            return ("✅", str(e))
        else:
            return ("❌", str(e))

def get(url: str, key: Optional[str] = None, extract: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.get(url, headers=headers, timeout=10)
        if not (200 <= resp.status_code < 300):
            if should_fail:
                return ("✅", f"状态码异常: {resp.status_code}")
            else:
                return ("❌", f"状态码异常: {resp.status_code}")
        try:
            json_data = resp.json()
        except ValueError:
            if should_fail:
                return ("✅", "响应不是有效的JSON格式")
            else:
                return ("❌", "响应不是有效的JSON格式")
        if extract:
            if extract not in json_data:
                if should_fail:
                    return ("✅", f"JSON中找不到属性: {extract}")
                else:
                    return ("❌", f"JSON中找不到属性: {extract}")
            result = json_data[extract]
        else:
            result = json_data
        if should_fail:
            return ("❌", "期望失败但成功")
        else:
            return ("✅", result)
    except Exception as e:
        if should_fail:
            return ("✅", str(e))
        else:
            return ("❌", str(e))

if __name__ == "__main__":
    run_test("创建string类型的桶", post("http://localhost:5090/bucket/test-string/string"))
    run_test("创建seq类型的桶", post("http://localhost:5090/bucket/test-seq/seq"))
    run_test("创建time类型的桶", post("http://localhost:5090/bucket/test-time/time"))
    run_test("创建test桶", post("http://localhost:5090/bucket/test/seq"))

    run_test("查看所有的桶", get("http://localhost:5090/bucket"))

    run_test("修改桶名", put("http://localhost:5090/bucket/test/test-new"))

    run_test("再次查看所有的桶", get("http://localhost:5090/bucket"))

    run_test("获取桶的类型", get("http://localhost:5090/bucket/type"))

    run_test("删除桶", delete("http://localhost:5090/bucket/test-new"))

    run_test("最后一次查看所有的桶", get("http://localhost:5090/bucket"))

    run_test("向string类型的桶插入数据_1",
             post("http://localhost:5090/kv",
                  body='''{
                             "Bucket": "test-string",
                             "Key": "test-key",
                             "Value": "test-value-1",
                             "Update": true
                           }'''))

    run_test("读取test-string表的所以数据_1", get("http://localhost:5090/kv/all/test-string"))

    run_test("向string类型的桶插入数据_2",
             post("http://localhost:5090/kv",
                  body='''{
                             "Bucket": "test-string",
                             "Key": "test-key",
                             "Value": "test-value-2",
                             "Update": true
                           }'''))

    run_test("读取test-string表的所以数据_2", get("http://localhost:5090/kv/all/test-string"))

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
                             "Key": "test-key",
                             "Value": "test-value"
                           }'''))

    run_test("读取test-seq表的所以数据", get("http://localhost:5090/kv/all/test-seq"))

    run_test("向time类型的桶插入数据",
             post("http://localhost:5090/kv",
                  body='''{
                             "Bucket": "test-time",
                             "Key": "test-key",
                             "Value": "test-value"
                           }'''))

    run_test("读取test-time表的所以数据", get("http://localhost:5090/kv/all/test-time"))
    
