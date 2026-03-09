#!/usr/bin/env python3
"""
HELM × Microsoft Agent Framework — Python Example

Demonstrates governed tool execution with HELM, producing receipts
and exporting an EvidencePack that can be verified offline.

Usage:
    # Start HELM first
    helm onboard --yes
    helm server &

    # Run this example
    python example.py
"""

import sys
import os

# Add the SDK to path
sys.path.insert(0, os.path.join(os.path.dirname(__file__), '..', '..', '..', 'sdk', 'python'))

from microsoft_agents import HelmAgentToolWrapper


def main():
    # Create a HELM-governed tool wrapper
    wrapper = HelmAgentToolWrapper(
        helm_url="http://localhost:8080",
        fail_closed=False,  # For demo: allow through even if HELM is unreachable
        metadata={
            "org": "contoso",
            "env": "demo",
            "framework": "ms-agent-framework",
        },
    )

    print("=" * 60)
    print("HELM × Microsoft Agent Framework — Example")
    print("=" * 60)
    print()

    # Simulate a multi-step agent workflow
    steps = [
        ("search_documents", {"query": "quarterly revenue report", "limit": 10}),
        ("analyze_data", {"dataset": "revenue_q4", "metrics": ["growth", "margin"]}),
        ("generate_summary", {"format": "executive_brief", "length": "short"}),
        ("send_email", {"to": "cfo@contoso.com", "subject": "Q4 Revenue Summary"}),
        ("delete_production_db", {"confirm": True}),  # This should be denied in real use
    ]

    for tool_name, args in steps:
        result = wrapper.execute(tool_name, args, principal="finance-agent")
        verdict_icon = "✅" if result.allowed else "❌"
        print(f"  {verdict_icon} {tool_name}: {result.receipt.verdict} "
              f"(L={result.receipt.lamport_clock})")

    print()
    print(f"Total receipts: {len(wrapper.receipts)}")
    print(f"Final hash:     {wrapper.receipts[-1].hash[:32]}...")

    # Export EvidencePack
    pack_path = "ms_agent_evidence.tar"
    pack_hash = wrapper.export_evidence_pack(pack_path)
    print(f"\nEvidencePack:   {pack_path}")
    print(f"Pack SHA-256:   {pack_hash[:32]}...")

    # Verify
    print(f"\nVerify with:    helm verify --bundle {pack_path}")
    print()


if __name__ == "__main__":
    main()
