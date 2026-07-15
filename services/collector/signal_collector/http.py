import json
from typing import Any, Mapping, Protocol
from urllib.parse import urlencode
from urllib.request import Request, urlopen


class HTTPClient(Protocol):
    def get_text(self, url: str, headers: Mapping[str, str] | None = None) -> str: ...

    def get_json(self, url: str, headers: Mapping[str, str] | None = None) -> Any: ...

    def post_form(
        self, url: str, payload: Mapping[str, str], headers: Mapping[str, str] | None = None
    ) -> dict: ...

    def post_json(
        self, url: str, payload: dict, headers: Mapping[str, str] | None = None
    ) -> dict: ...


class UrllibHTTPClient:
    """Standard-library HTTP client; adapters own every URL they call."""

    def get_text(self, url: str, headers: Mapping[str, str] | None = None) -> str:
        request = Request(url, headers=self._headers(headers))
        with urlopen(request, timeout=20) as response:  # noqa: S310 - adapter-owned URL
            return response.read().decode("utf-8", errors="replace")

    def get_json(self, url: str, headers: Mapping[str, str] | None = None) -> dict:
        return json.loads(self.get_text(url, headers))

    def post_json(
        self, url: str, payload: dict, headers: Mapping[str, str] | None = None
    ) -> dict:
        request_headers = self._headers(headers)
        request_headers["Content-Type"] = "application/json"
        request = Request(
            url,
            data=json.dumps(payload).encode("utf-8"),
            headers=request_headers,
            method="POST",
        )
        with urlopen(request, timeout=20) as response:  # noqa: S310 - configured backend URL
            return json.loads(response.read().decode("utf-8"))

    def post_form(
        self, url: str, payload: Mapping[str, str], headers: Mapping[str, str] | None = None
    ) -> dict:
        request_headers = self._headers(headers)
        request_headers["Content-Type"] = "application/x-www-form-urlencoded"
        request = Request(
            url,
            data=urlencode(payload).encode("utf-8"),
            headers=request_headers,
            method="POST",
        )
        with urlopen(request, timeout=20) as response:  # noqa: S310 - adapter-owned OAuth URL
            return json.loads(response.read().decode("utf-8"))

    @staticmethod
    def _headers(headers: Mapping[str, str] | None) -> dict[str, str]:
        result = {
            "Accept": "application/json, text/html;q=0.9",
            "User-Agent": "ai-signal-radar/0.1 (+personal research collector)",
        }
        if headers:
            result.update(headers)
        return result
