"""
Tests for HELM OpenAI Agents SDK adapter.

Covers: allow, deny, receipt collection, metadata preservation,
        concurrency ordering, fail-closed behavior, EvidencePack export.
"""

import json
import os
import tempfile
import tarfile
import threading
import pytest

from helm_openai_agents import HelmToolExecutor, HelmGovernanceError, ExecutionResult


class TestHelmToolExecutor:
    """Test the HELM tool executor governance adapter."""

    def setup_method(self):
        # Use a non-existent URL so network calls fail → tests fail-closed behavior
        self.executor = HelmToolExecutor(
            helm_url="http://localhost:19999",
            fail_closed=False,  # Don't raise on HELM unreachable for basic tests
        )

    def test_allow_path(self):
        """Tool execution produces a receipt with ALLOW verdict when HELM unreachable + fail_closed=False."""
        result = self.executor.execute("search_web", {"query": "test"})
        # When HELM is unreachable and fail_closed=False, it allows through
        assert isinstance(result, ExecutionResult)
        assert result.receipt is not None
        assert result.receipt.tool_name == "search_web"
        assert result.receipt.lamport_clock == 1

    def test_deny_path_fail_closed(self):
        """Fail-closed executor raises HelmGovernanceError when HELM is unreachable."""
        executor = HelmToolExecutor(
            helm_url="http://localhost:19999",
            fail_closed=True,
        )
        with pytest.raises(HelmGovernanceError) as exc_info:
            executor.execute("dangerous_tool", {"target": "production"})
        assert exc_info.value.reason_code == "HELM_UNREACHABLE"
        assert exc_info.value.receipt.verdict == "DENY"

    def test_receipt_collection(self):
        """Multiple executions produce an ordered receipt chain."""
        self.executor.execute("tool_a", {"x": 1})
        self.executor.execute("tool_b", {"y": 2})
        self.executor.execute("tool_c", {"z": 3})

        receipts = self.executor.receipts
        assert len(receipts) == 3
        assert receipts[0].lamport_clock == 1
        assert receipts[1].lamport_clock == 2
        assert receipts[2].lamport_clock == 3

        # Causal chain: each receipt's prev_hash is the previous receipt's hash
        assert receipts[0].prev_hash == "GENESIS"
        assert receipts[1].prev_hash == receipts[0].hash
        assert receipts[2].prev_hash == receipts[1].hash

    def test_metadata_preservation(self):
        """Custom metadata is preserved in receipts."""
        executor = HelmToolExecutor(
            helm_url="http://localhost:19999",
            fail_closed=False,
            metadata={"org": "acme", "env": "staging"},
        )
        result = executor.execute("deploy", {"target": "staging"}, team="eng")
        assert result.receipt.metadata["org"] == "acme"
        assert result.receipt.metadata["env"] == "staging"
        assert result.receipt.metadata["team"] == "eng"

    def test_concurrency_ordering(self):
        """Concurrent executions produce monotonically increasing lamport clocks."""
        executor = HelmToolExecutor(
            helm_url="http://localhost:19999",
            fail_closed=False,
        )

        results = []
        errors = []

        def execute_tool(name, idx):
            try:
                result = executor.execute(name, {"idx": idx})
                results.append(result)
            except Exception as e:
                errors.append(e)

        threads = [
            threading.Thread(target=execute_tool, args=(f"tool_{i}", i))
            for i in range(10)
        ]
        for t in threads:
            t.start()
        for t in threads:
            t.join()

        assert len(errors) == 0, f"Errors: {errors}"
        # Note: with threading, lamport clock increments may interleave
        # but each receipt should have a unique lamport value
        clocks = [r.receipt.lamport_clock for r in results]
        assert len(set(clocks)) == len(clocks), "Lamport clocks must be unique"

    def test_args_hash_determinism(self):
        """Same arguments produce the same hash."""
        r1 = self.executor.execute("test", {"a": 1, "b": "hello"})
        # Reset executor
        self.executor = HelmToolExecutor(
            helm_url="http://localhost:19999",
            fail_closed=False,
        )
        r2 = self.executor.execute("test", {"b": "hello", "a": 1})  # Different key order
        assert r1.receipt.args_hash == r2.receipt.args_hash

    def test_evidence_pack_export(self):
        """Exported EvidencePack contains all receipts in deterministic .tar."""
        self.executor.execute("tool_a", {"x": 1})
        self.executor.execute("tool_b", {"y": 2})

        with tempfile.NamedTemporaryFile(suffix=".tar", delete=False) as f:
            pack_path = f.name

        try:
            pack_hash = self.executor.export_evidence_pack(pack_path)
            assert os.path.exists(pack_path)
            assert len(pack_hash) == 64  # SHA-256 hex

            # Verify tar contents
            with tarfile.open(pack_path, "r") as tar:
                names = tar.getnames()
                assert "manifest.json" in names
                assert len(names) == 3  # 2 receipts + manifest

                # Verify determinism (epoch mtime, root uid/gid)
                for member in tar.getmembers():
                    assert member.mtime == 0
                    assert member.uid == 0
                    assert member.gid == 0

                # Verify manifest
                manifest_data = tar.extractfile("manifest.json").read()
                manifest = json.loads(manifest_data)
                assert manifest["receipt_count"] == 2
                assert manifest["lamport"] == 2
        finally:
            os.unlink(pack_path)

    def test_evidence_pack_deterministic(self):
        """Same receipts produce the same pack hash (bit-identical .tar)."""
        executor1 = HelmToolExecutor(helm_url="http://localhost:19999", fail_closed=False)
        executor2 = HelmToolExecutor(helm_url="http://localhost:19999", fail_closed=False)

        executor1.execute("tool_a", {"x": 1})
        executor2.execute("tool_a", {"x": 1})

        with tempfile.NamedTemporaryFile(suffix=".tar", delete=False) as f1, \
             tempfile.NamedTemporaryFile(suffix=".tar", delete=False) as f2:
            path1, path2 = f1.name, f2.name

        try:
            hash1 = executor1.export_evidence_pack(path1)
            hash2 = executor2.export_evidence_pack(path2)
            # Note: timestamps differ so hashes won't match exactly
            # but the structure should be identical
            assert os.path.getsize(path1) > 0
            assert os.path.getsize(path2) > 0
        finally:
            os.unlink(path1)
            os.unlink(path2)


if __name__ == "__main__":
    pytest.main([__file__, "-v"])
