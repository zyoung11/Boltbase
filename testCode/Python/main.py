import requests
import json
from typing import Optional, Any, Tuple
from rich.console import Console
from rich.panel import Panel
from rich.syntax import Syntax

def _get_status_color(status_code: int) -> str:
    if 200 <= status_code < 300:
        return "green"
    if 400 <= status_code < 500:
        return "yellow"
    if 500 <= status_code < 600:
        return "red"
    return "white"


def run_test(description: str, response: Tuple[str, Any, int], extract: Optional[str] = None) -> Any:
    console = Console()
    status, content, status_code = response
    color = _get_status_color(status_code)

    title = f"""{description}: {status} [bold {color}]HTTP {status_code}[/bold {color}]"""

    display_content = content
    if extract is None and isinstance(content, dict) and 'buckets' in content:
        display_content = content['buckets']

    if isinstance(display_content, dict) or isinstance(display_content, list):
        json_str = json.dumps(display_content, indent=4, ensure_ascii=False)
        body = Syntax(json_str, "json", theme="dracula", line_numbers=True, background_color="default")
    else:
        body = str(display_content)

    console.print(
        Panel(
            body,
            title=title,
            border_style="blue",
            expand=True,
        )
    )
    console.print()

    if extract:
        if isinstance(content, dict) and extract in content:
            return content[extract]
        else:
            console.print(f"[bold red]Warning:[/bold red] Could not extract '{extract}' from response.")
            return None
    return None


def post(url: str, body: Optional[str] = None, key: Optional[str] = None,
         should_fail: bool = False) -> Tuple[str, Any, int]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    try:
        resp = requests.post(url, **kwargs)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code
            else:
                try:
                    return "✅", resp.json(), status_code
                except ValueError:
                    return "✅", {"response": resp.text}, status_code
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code
            else:
                return "❌", f"状态码异常: {status_code}", status_code
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999


def delete(url: str, key: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any, int]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.delete(url, headers=headers, timeout=10)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code
            else:
                try:
                    return "✅", resp.json(), status_code
                except ValueError:
                    return "✅", {"response": resp.text}, status_code
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code
            else:
                return "❌", f"状态码异常: {status_code}", status_code
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999


def put(url: str, body: Optional[str] = None, key: Optional[str] = None, should_fail: bool = False) -> Tuple[
    str, Any, int]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": headers, "timeout": 10}
    if body:
        kwargs["data"] = body
    try:
        resp = requests.put(url, **kwargs)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code
            else:
                try:
                    return "✅", resp.json(), status_code
                except ValueError:
                    return "✅", {"response": resp.text}, status_code
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code
            else:
                return "❌", f"状态码异常: {status_code}", status_code
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999


def get(url: str, key: Optional[str] = None, should_fail: bool = False) -> Tuple[str, Any, int]:
    headers = {"Content-Type": "application/json"}
    if key:
        headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.get(url, headers=headers, timeout=10)
        status_code = resp.status_code
        if not (200 <= status_code < 300):
            if should_fail:
                return "✅", f"状态码异常: {status_code}", status_code
            else:
                return "❌", f"状态码异常: {status_code}", status_code
        try:
            json_data = resp.json()
        except ValueError:
            return ("❌" if not should_fail else "✅"), "响应不是有效的JSON格式", status_code

        if should_fail:
            return "❌", "期望失败但成功", status_code
        else:
            return "✅", json_data, status_code
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999

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
