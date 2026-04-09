from __future__ import annotations

from datetime import datetime, timezone
from urllib.parse import urlparse

from agentctl_sdk.models.action import Action

from .client import AgentctlClient


class AgentctlPythonGuard:
    def __init__(self, client: AgentctlClient, default_context: dict | None = None) -> None:
        self.client = client
        self.default_context = default_context or {}

    def wrap_exec(self, exec_fn, context: dict | None = None):
        def guarded(command: str):
            self._assert_allowed(
                Action.RUN_CODE,
                {"language": "bash", "command": command},
                f"execute command: {command}",
                context,
            )
            return exec_fn(command)

        return guarded

    def wrap_write_file(self, write_fn, context: dict | None = None):
        def guarded(path: str, content: str):
            self._assert_allowed(
                Action.WRITE_FILE,
                {
                    "path": path,
                    "operation": "overwrite",
                    "size_bytes": len(content.encode("utf-8")),
                },
                f"write file: {path}",
                context,
            )
            return write_fn(path, content)

        return guarded

    def wrap_fetch(self, fetch_fn, context: dict | None = None):
        def guarded(url: str, method: str = "GET"):
            parsed = urlparse(url)
            self._assert_allowed(
                Action.CALL_EXTERNAL_API,
                {"url": url, "method": method, "domain": parsed.hostname},
                f"call external API: {parsed.hostname}",
                context,
            )
            return fetch_fn(url, method=method)

        return guarded

    def _assert_allowed(self, action: Action, params: dict, reason: str, context: dict | None) -> None:
        request_context = self._build_context(context)
        decision = self.client.gate(
            action=action,
            params=params,
            reason=reason,
            context=request_context,
        )
        if decision.verdict != "allow":
            raise RuntimeError(f"agentctl {decision.verdict}: {decision.reason}")

    def _build_context(self, context: dict | None) -> dict:
        merged = {
            **self.default_context,
            **(context or {}),
        }
        merged.setdefault("session_id", f"py-{int(datetime.now(tz=timezone.utc).timestamp() * 1000)}")
        merged.setdefault("timestamp", datetime.now(tz=timezone.utc))
        return merged
