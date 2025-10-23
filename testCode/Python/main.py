import requests
from typing import Dict, Optional, Any, Tuple

def post(url: str, body: Optional[str] = None, key: Optional[str] = None) -> Dict:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    resp = requests.post(url, **kwargs)
    if 200 <= resp.status_code < 300:
        print("\n✅")
        try:
            return resp.json()
        except ValueError:
            return {"response": resp.text}
    else:
        print("\n❌")
        raise RuntimeError(f"状态码异常: {resp.status_code}")

def delete(url: str, key: Optional[str] = None) -> Dict:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    resp = requests.delete(url, headers=headers, timeout=10)
    if 200 <= resp.status_code < 300:
        print("\n✅")
        try:
            return resp.json()
        except ValueError:
            return {"response": resp.text}
    else:
        print("\n❌")
        raise RuntimeError(f"状态码异常: {resp.status_code}")

def put(url: str, body: Optional[str] = None, key: Optional[str] = None) -> Dict:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    resp = requests.put(url, **kwargs)
    if 200 <= resp.status_code < 300:
        print("\n✅")
        try:
            return resp.json()
        except ValueError:
            return {"response": resp.text}
    else:
        print("\n❌")
        raise RuntimeError(f"状态码异常: {resp.status_code}")

def get(url: str, key: Optional[str] = None, extract: Optional[str] = None) -> Tuple[str, Any]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.get(url, headers=headers, timeout=10)
        if not (200 <= resp.status_code < 300):
            return ("❌", f"状态码异常: {resp.status_code}")
        try:
            json_data = resp.json()
        except ValueError:
            return ("❌", "响应不是有效的JSON格式")
        if extract:
            if extract not in json_data:
                return ("❌", f"JSON中找不到属性: {extract}")
            result = json_data[extract]
        else:
            result = json_data
        return ("✅", result)
    except Exception as e:
        return ("❌", str(e))

if __name__ == "__main__":

    # 例子
    # print("POST:", 
    #       post("https://httpbin.org/post",
    #            body='{"name":"Alice","age":18}',
    #            key="sk-1234567890abcdef"))

    # print("DELETE:", 
    #       delete("https://httpbin.org/delete",
    #              key="sk-123456789f"))
    
    # print("PUT:",
    #       put("https://httpbin.org/put",
    #           body='{"name":"Charlie","age":25}',
    #           key="sk-1234567890abcdef"))

    # status, content = get("http://localhost:5090",
    #                       key="sk-2312312312f",
    #                       extract="token")
    # print(f"GET: {status}, 内容: {content}")
    
    print("创建string类型的桶：",
          post("http://localhost:5090/bucket/test-string/string"))

    print("创建seq类型的桶：",
          post("http://localhost:5090/bucket/test-seq/seq"))

    print("创建time类型的桶：",
          post("http://localhost:5090/bucket/test-time/time"))

    print("创建test桶：",
          post("http://localhost:5090/bucket/test/seq"))

    print("\n查看所以的桶：")
    print(get("http://localhost:5090/bucket"))

    print("修改桶名：",
          put("http://localhost:5090/bucket/test/test-new"))

    print("\n查看所以的桶：")
    print(get("http://localhost:5090/bucket"))

    print("\n获取桶的类型：")
    print(get("http://localhost:5090/bucket/type"))

    print("删除桶：",
          delete("http://localhost:5090/bucket/test-new"))

    print("\n查看所以的桶：")
    print(get("http://localhost:5090/bucket"))




        
