# HELM SDK — Java

Typed Java client for the HELM kernel API. Uses `java.net.http` (JDK 17+) and Gson.

## Install (Maven)

```xml
<dependency>
    <groupId>ai.mindburn.helm</groupId>
    <artifactId>helm-sdk</artifactId>
    <version>0.1.0</version>
</dependency>
```

## Quick Example

```java
import labs.mindburn.helm.HelmClient;
import labs.mindburn.helm.TypesGen.*;

var helm = new HelmClient("http://localhost:8080");

// Chat completions via HELM proxy
var req = new ChatCompletionRequest();
req.model = "gpt-4";
req.messages = List.of(new ChatMessage("user", "List files in /tmp"));

try {
    var res = helm.chatCompletions(req);
    System.out.println(res.choices.get(0).message.content);
} catch (HelmClient.HelmApiException e) {
    System.out.println("Denied: " + e.reasonCode);
}

// Conformance
var conf = helm.conformanceRun(new ConformanceRequest("L2"));
System.out.println(conf.verdict + " " + conf.gates + " gates");
```

## API

| Method | Endpoint |
|--------|----------|
| `chatCompletions(req)` | `POST /v1/chat/completions` |
| `approveIntent(req)` | `POST /api/v1/kernel/approve` |
| `listSessions()` | `GET /api/v1/proofgraph/sessions` |
| `getReceipts(sessionId)` | `GET /api/v1/proofgraph/sessions/{id}/receipts` |
| `conformanceRun(req)` | `POST /api/v1/conformance/run` |
| `health()` | `GET /healthz` |
| `version()` | `GET /version` |
