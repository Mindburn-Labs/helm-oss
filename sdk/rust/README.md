# HELM SDK — Rust

Typed Rust client for the HELM kernel API. Deps: `reqwest` + `serde`.

## Install

```toml
[dependencies]
helm-sdk = { git = "https://github.com/Mindburn-Labs/helm-oss", path = "sdk/rust" }
```

Or when published:
```bash
cargo add helm-sdk
```

## Quick Example

```rust
use helm_sdk::{HelmClient, ChatCompletionRequest, ChatMessage, ConformanceRequest};

fn main() {
    let client = HelmClient::new("http://localhost:8080");

    // Chat completions via HELM proxy
    let res = client.chat_completions(&ChatCompletionRequest {
        model: "gpt-4".into(),
        messages: vec![ChatMessage {
            role: "user".into(),
            content: "List files in /tmp".into(),
            tool_call_id: None,
        }],
        ..Default::default()
    });

    match res {
        Ok(r) => println!("{:?}", r.choices[0].message.content),
        Err(e) => println!("Denied: {:?}", e.reason_code),
    }

    // Conformance
    let conf = client.conformance_run(&ConformanceRequest {
        level: "L2".into(),
        profile: None,
    }).unwrap();
    println!("{} {} gates", conf.verdict, conf.gates);
}
```
