# HELM LangGraph Shim (Example)
from langgraph.graph import StateGraph
from helm_sdk import HelmClient

# Initialize HELM Client as the PEP
helm = HelmClient(base_url="http://localhost:8080/v1")

def helm_governed_tool(state):
    # This node wraps tool calls in a HELM transaction
    intent = state.get("intent")
    receipt = helm.execute_tool(intent)
    return {"receipt": receipt}

# Build Graph
builder = StateGraph(dict)
builder.add_node("governor", helm_governed_tool)
# ... rest of graph
