# HELM OpenAI Agents SDK Shim (Example)
from openai import OpenAI
# The simplest shim: 1-line base_url swap
client = OpenAI(base_url="http://localhost:8080/v1")

# All agent actions now produce signed receipts automatically
response = client.chat.completions.create(
    model="gpt-4",
    messages=[{"role": "user", "content": "Trigger governed tool"}]
)
print(f"Verified Receipt: {response.headers.get('X-Helm-Receipt-ID')}")
