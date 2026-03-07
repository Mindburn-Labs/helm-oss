# HELM for Humans

If you just want to stop your AI from doing things it shouldn't, this is for you. No UCS/UCS jargon, just the facts.

### 1. What does HELM actually do?
Imagine your AI agent has a credit card and access to your terminal. HELM is the person standing over its shoulder. 
- If the AI tries to buy a $10,000 server, HELM stops it (Budgeting).
- If the AI tries to run `rm -rf /`, HELM stops it (Sandboxing).
- If the AI tries to call a tool you haven't approved, HELM stops it (Fail-Closed).

### 2. How do I install it?
```bash
curl -fsSL https://raw.githubusercontent.com/Mindburn-Labs/helm-oss/main/install.sh | bash
helm server
```
That's it. It starts a local server. You then point your Python/Node.js OpenAI client to `http://localhost:8080/v1`.

### 3. Do I need a database?
Nope. By default, it uses a local file (`data/helm.db`) so you can try it out instantly. For big companies, we support Postgres.

### 4. What is a "Receipt"?
Every time your AI does something, HELM signs a digital "receipt". 
- **The AI says**: "I'm going to read this file."
- **HELM says**: "Allowed. Here is your signed receipt #42."
You can use these receipts later to prove exactly what happened if something goes wrong.

### 5. Does it slow down my AI?
Barely. HELM adds about **5-10ms** of overhead per request. For a model that takes 2 seconds to think, this is invisible.

### 6. Is it free?
Yes, it's Open Source (BSL-1.1). You can use it in production for free. We only charge if you want to sell HELM as a managed service to others.

---
**Ready to go?** Head back to the [Quickstart](../README.md#quickstart).
