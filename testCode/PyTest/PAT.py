import requests
import json
from typing import Optional, Any, Tuple, Dict, Union
from rich.console import Console
from rich.panel import Panel
from rich.syntax import Syntax
from rich.table import Table
from rich import box

def _get_status_color(status_code: int) -> str:
    if 200 <= status_code < 300:
        return "green"
    if 400 <= status_code < 500:
        return "yellow"
    if 500 <= status_code < 600:
        return "red"
    return "white"

def print_info(title: str, info: Dict[str, Any]):
    console = Console()

    table = Table(
        show_header=True,
        header_style="magenta", 
        box=box.ROUNDED,
        expand=True
    )
    table.add_column("Key", style="dim", width=20)
    table.add_column("Value")

    for k, v in info.items():
        table.add_row(str(k), str(v))

    console.print(Panel(table, title=title, border_style="green", expand=True))

def _deep_get(obj: Any, path: str) -> Any:
    keys = path.split(".")
    cur = obj
    for k in keys:
        if isinstance(cur, dict):
            cur = cur.get(k)
        elif isinstance(cur, list) and k.isdigit():
            cur = cur[int(k)]
        else:
            return None
        if cur is None:
            break
    return cur

def run_test(
    description: str,
    response: Tuple[str, Any, int, Optional[str]],
    *extract_paths: str
) -> Any:
    console = Console()
    status, content, status_code, _ = response
    color = _get_status_color(status_code)

    title = f"""{description}: {status} [bold {color}]HTTP {status_code}[/bold {color}]"""

    display_content = content
    if not extract_paths and isinstance(content, dict) and 'buckets' in content:
        display_content = content['buckets']

    if isinstance(display_content, (dict, list)):
        json_str = json.dumps(display_content, indent=4, ensure_ascii=False)
        body = Syntax(json_str, "json", theme="dracula", line_numbers=True, background_color="default")
    else:
        body = str(display_content)

    console.print(Panel(body, title=title, border_style="blue", expand=True))
    console.print()

    if not extract_paths:
        return None
    if len(extract_paths) == 1:
        value = _deep_get(content, extract_paths[0])
        if value is None:
            console.print(f"[bold red]Warning:[/bold red] Could not extract '{extract_paths[0]}' from response.")
        return value
    values = []
    for path in extract_paths:
        v = _deep_get(content, path)
        if v is None:
            console.print(f"[bold red]Warning:[/bold red] Could not extract '{path}' from response.")
        values.append(v)
    return tuple(values)

def post(url: str, body: Optional[Union[str, Dict[str, Any], list]] = None, key: Optional[str] = None,
         should_fail: bool = False, extract: Optional[str] = None, headers: Optional[Dict[str, str]] = None) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": request_headers, "timeout": 10}
    if body is not None:
        if isinstance(body, (dict, list)):
            kwargs["data"] = json.dumps(body, ensure_ascii=False)
        else:
            kwargs["data"] = body
    try:
        resp = requests.post(url, **kwargs)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code, extract
            else:
                try:
                    return "✅", resp.json(), status_code, extract
                except ValueError:
                    return "✅", {"response": resp.text}, status_code, extract
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code, extract
            else:
                return "❌", f"状态码异常: {status_code}", status_code, extract
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract


def delete(url: str, key: Optional[str] = None, should_fail: bool = False, extract: Optional[str] = None, headers: Optional[Dict[str, str]] = None) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.delete(url, headers=request_headers, timeout=10)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code, extract
            else:
                try:
                    return "✅", resp.json(), status_code, extract
                except ValueError:
                    return "✅", {"response": resp.text}, status_code, extract
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code, extract
            else:
                return "❌", f"状态码异常: {status_code}", status_code, extract
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract


def put(url: str, body: Optional[Union[str, Dict[str, Any], list]] = None, key: Optional[str] = None, should_fail: bool = False, extract: Optional[str] = None, headers: Optional[Dict[str, str]] = None) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"
    kwargs = {"headers": request_headers, "timeout": 10}
    if body is not None:
        if isinstance(body, (dict, list)):
            kwargs["data"] = json.dumps(body, ensure_ascii=False)
        else:
            kwargs["data"] = body
    try:
        resp = requests.put(url, **kwargs)
        status_code = resp.status_code
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code, extract
            else:
                try:
                    return "✅", resp.json(), status_code, extract
                except ValueError:
                    return "✅", {"response": resp.text}, status_code, extract
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code, extract
            else:
                return "❌", f"状态码异常: {status_code}", status_code, extract
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract


def get(url: str, key: Optional[str] = None, should_fail: bool = False, extract: Optional[str] = None, headers: Optional[Dict[str, str]] = None) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"
    try:
        resp = requests.get(url, headers=request_headers, timeout=10)
        status_code = resp.status_code
        if not (200 <= status_code < 300):
            if should_fail:
                return "✅", f"状态码异常: {status_code}", status_code, extract
            else:
                return "❌", f"状态码异常: {status_code}", status_code, extract
        try:
            json_data = resp.json()
        except ValueError:
            return ("❌" if not should_fail else "✅"), "响应不是有效的JSON格式", status_code, extract

        if should_fail:
            return "❌", "期望失败但成功", status_code, extract
        else:
            return "✅", json_data, status_code, extract
    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract


def patch(url: str,
          body: Optional[Union[str, Dict[str, Any], list]] = None,
          key: Optional[str] = None,
          should_fail: bool = False,
          extract: Optional[str] = None,
          headers: Optional[Dict[str, str]] = None
          ) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"

    kwargs = {"headers": request_headers, "timeout": 10}
    if body is not None:
        if isinstance(body, (dict, list)):
            kwargs["data"] = json.dumps(body, ensure_ascii=False)
        else:
            kwargs["data"] = body
    try:
        resp = requests.patch(url, **kwargs)
        status_code = resp.status_code

        # 2xx 算成功
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code, extract
            try:
                return "✅", resp.json(), status_code, extract
            except ValueError:
                return "✅", {"response": resp.text}, status_code, extract
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code, extract
            return "❌", f"状态码异常: {status_code}", status_code, extract

    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract


def option(url: str,
           key: Optional[str] = None,
           should_fail: bool = False,
           extract: Optional[str] = None,
           headers: Optional[Dict[str, str]] = None
           ) -> Tuple[str, Any, int, Optional[str]]:
    request_headers = {"Content-Type": "application/json"}
    if headers:
        request_headers.update(headers)
    if key:
        request_headers["Authorization"] = f"Bearer {key}"

    try:
        resp = requests.options(url, headers=request_headers, timeout=10)
        status_code = resp.status_code

        # 2xx 算成功
        if 200 <= status_code < 300:
            if should_fail:
                return "❌", f"期望失败但成功: {status_code}", status_code, extract
            try:
                # OPTIONS 多数返回空 body，若服务端返回 JSON 也能解析
                json_data = resp.json()
            except ValueError:
                # 无 body 时把响应头里常用 CORS 信息带回来即可
                json_data = {
                    "allow": resp.headers.get("Allow"),
                    "access_control_allow_methods": resp.headers.get("Access-Control-Allow-Methods"),
                    "access_control_allow_headers": resp.headers.get("Access-Control-Allow-Headers"),
                    "access_control_max_age": resp.headers.get("Access-Control-Max-Age"),
                }
            return "✅", json_data, status_code, extract
        else:
            if should_fail:
                return "✅", {"error": f"状态码异常: {status_code}"}, status_code, extract
            return "❌", f"状态码异常: {status_code}", status_code, extract

    except Exception as e:
        return ("❌" if not should_fail else "✅"), str(e), 999, extract
