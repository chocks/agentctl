from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from typing import Any

from agentctl_sdk.api.gate_api import GateApi
from agentctl_sdk.api.replay_api import ReplayApi
from agentctl_sdk.api.trace_api import TraceApi
from agentctl_sdk.api_client import ApiClient
from agentctl_sdk.configuration import Configuration
from agentctl_sdk.models.action import Action
from agentctl_sdk.models.gate_request import GateRequest
from agentctl_sdk.models.replay_request import ReplayRequest
from agentctl_sdk.models.request_context import RequestContext


@dataclass
class AgentctlClient:
    base_url: str = "http://127.0.0.1:8080"
    token: str | None = None
    default_context: dict[str, Any] | None = None

    def __post_init__(self) -> None:
        configuration = Configuration(host=self.base_url, access_token=self.token)
        self.api_client = ApiClient(configuration=configuration)
        self.gate_api = GateApi(self.api_client)
        self.trace_api = TraceApi(self.api_client)
        self.replay_api = ReplayApi(self.api_client)

    def gate(
        self,
        action: Action,
        params: dict[str, Any],
        reason: str,
        context: dict[str, Any] | None = None,
        headers: dict[str, Any] | None = None,
    ):
        request = GateRequest(
            action=action,
            params=params,
            reason=reason,
            context=self._build_context(context),
        )
        return self.gate_api.gate_action(request, _headers=headers)

    def list_traces(
        self,
        session_id: str | None = None,
        action: Action | None = None,
        verdict: str | None = None,
        package_name: str | None = None,
        since: datetime | None = None,
        until: datetime | None = None,
        limit: int | None = None,
    ):
        return self.trace_api.list_traces(
            session_id=session_id,
            action=action,
            verdict=verdict,
            package=package_name,
            since=since,
            until=until,
            limit=limit,
        )

    def replay(
        self,
        session_id: str,
        policy_path: str | None = None,
        limit: int | None = None,
    ):
        request = ReplayRequest(
            session_id=session_id,
            policy_path=policy_path,
            limit=limit,
        )
        return self.replay_api.replay_session(request)

    def _build_context(self, context: dict[str, Any] | None) -> RequestContext | None:
        merged = {
            **(self.default_context or {}),
            **(context or {}),
        }
        if "session_id" not in merged or "timestamp" not in merged:
            return None
        return RequestContext(**merged)
