"""Tests for HELM MS Agent Framework adapter."""

import json
import os
import tempfile
import tarfile
import threading
import pytest

from helm_ms_agent import HelmAgentToolWrapper, GovernedResult


class TestHelmAgentToolWrapper:

    def setup_method(self):
        self.wrapper = HelmAgentToolWrapper(
            helm_url="http://localhost:19999",
            fail_closed=False,
        )

    def test_allow_path(self):
        result = self.wrapper.execute("search", {"query": "test"})
        assert isinstance(result, GovernedResult)
        assert result.receipt.tool_name == "search"
        assert result.receipt.lamport_clock == 1

    def test_deny_path_fail_closed(self):
        wrapper = HelmAgentToolWrapper(
            helm_url="http://localhost:19999",
            fail_closed=True,
        )
        result = wrapper.execute("dangerous", {"target": "prod"})
        assert not result.allowed
        assert result.receipt.verdict == "DENY"
        assert result.receipt.reason_code == "HELM_UNREACHABLE"

    def test_receipt_collection(self):
        self.wrapper.execute("a", {"x": 1})
        self.wrapper.execute("b", {"y": 2})
        self.wrapper.execute("c", {"z": 3})

        receipts = self.wrapper.receipts
        assert len(receipts) == 3
        assert receipts[0].prev_hash == "GENESIS"
        assert receipts[1].prev_hash == receipts[0].hash
        assert receipts[2].prev_hash == receipts[1].hash

    def test_metadata_preservation(self):
        wrapper = HelmAgentToolWrapper(
            helm_url="http://localhost:19999",
            fail_closed=False,
            metadata={"org": "contoso"},
        )
        result = wrapper.execute("deploy", {"env": "staging"}, team="platform")
        assert result.receipt.metadata["org"] == "contoso"
        assert result.receipt.metadata["team"] == "platform"
        assert result.receipt.metadata["framework"] == "ms-agent-framework"

    def test_concurrency_ordering(self):
        wrapper = HelmAgentToolWrapper(
            helm_url="http://localhost:19999",
            fail_closed=False,
        )
        results = []

        def run(name, idx):
            results.append(wrapper.execute(name, {"idx": idx}))

        threads = [threading.Thread(target=run, args=(f"t{i}", i)) for i in range(10)]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        clocks = [r.receipt.lamport_clock for r in results]
        assert len(set(clocks)) == len(clocks)

    def test_evidence_pack_export(self):
        self.wrapper.execute("a", {"x": 1})
        self.wrapper.execute("b", {"y": 2})

        with tempfile.NamedTemporaryFile(suffix=".tar", delete=False) as f:
            path = f.name

        try:
            pack_hash = self.wrapper.export_evidence_pack(path)
            assert os.path.exists(path)
            assert len(pack_hash) == 64

            with tarfile.open(path, "r") as tar:
                names = tar.getnames()
                assert "manifest.json" in names
                assert len(names) == 3
                for m in tar.getmembers():
                    assert m.mtime == 0
                    assert m.uid == 0
        finally:
            os.unlink(path)

    def test_wrap_tool_decorator(self):
        wrapper = HelmAgentToolWrapper(
            helm_url="http://localhost:19999",
            fail_closed=False,
        )
        call_log = []

        @wrapper.wrap_tool
        def my_tool(x=1):
            call_log.append(x)
            return x * 2

        result = my_tool(x=42)
        assert result == 84
        assert call_log == [42]
        assert len(wrapper.receipts) == 1


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
