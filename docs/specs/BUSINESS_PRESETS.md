# HELM Business Function Presets

> Ready-to-use governance presets for common business functions.
> Install with: `helm preset apply <name>`

## Preset Index

| Preset             | Function             | Risk Level | Policies                                              |
| ------------------ | -------------------- | ---------- | ----------------------------------------------------- |
| `engineering`      | Software Engineering | Medium     | Code execution, file access, API calls, deploy gates  |
| `it-sre`           | IT / SRE / DevOps    | High       | Infrastructure changes, incident response, monitoring |
| `security`         | Security Operations  | Critical   | Vuln scanning, pentest, threat response               |
| `product-research` | Product & Research   | Low        | Data analysis, prototyping, model experimentation     |
| `customer-support` | Customer Support     | Medium     | Data lookup, ticket ops, PII handling                 |
| `sales-revops`     | Sales & Revenue Ops  | Medium     | CRM access, pricing, deal approval                    |
| `marketing`        | Marketing            | Low        | Content creation, campaign management                 |
| `finance`          | Finance & Accounting | Critical   | Financial transactions, reporting, audit              |
| `hr`               | Human Resources      | High       | Employee data, payroll, compliance                    |
| `procurement`      | Procurement          | High       | Vendor management, contract execution                 |
| `legal-compliance` | Legal & Compliance   | Critical   | Contract review, regulatory filings                   |
| `operations`       | General Operations   | Medium     | Workflow automation, scheduling                       |

## Preset Format

```yaml
apiVersion: helm.sh/v1
kind: BusinessPreset
metadata:
  name: engineering
  version: 1.0.0
  description: Software engineering team governance preset
  function: engineering
  risk_level: medium

spec:
  capabilities:
    allowed:
      - file_read
      - file_write
      - code_execute
      - api_call
      - browse_fetch
    gated:
      - deploy_change_infra
      - transact_pay
    denied:
      - self_extend
      - communicate_external

  budget:
    daily_limit_cents: 50000 # $500/day
    per_action_limit_cents: 5000 # $50/action
    model_tier: standard

  approval_rules:
    - effect_type: deploy_change_infra
      require: manager_approval
      timeout: 1h
    - effect_type: file_write
      match:
        path_prefix: /etc/
      require: security_approval

  sandbox:
    mode: strict
    network: restricted
    filesystem: scoped
```

## Capability Taxonomy

| Capability            | Description                         | Default Risk |
| --------------------- | ----------------------------------- | ------------ |
| `plan`                | Planning and reasoning              | Low          |
| `approve`             | Approval workflows                  | Medium       |
| `execute`             | Code/script execution               | High         |
| `file_read`           | File system read access             | Low          |
| `file_write`          | File system write access            | Medium       |
| `browse_fetch`        | Web browsing and fetching           | Low          |
| `communicate`         | Send messages/emails                | Medium       |
| `transact_pay`        | Financial transactions              | Critical     |
| `deploy_change_infra` | Infrastructure changes              | Critical     |
| `analyze_report`      | Data analysis and reporting         | Low          |
| `negotiate_request`   | Negotiate or request approval       | Medium       |
| `self_extend`         | Install packages, modify own config | Critical     |
| `self_fix`            | Self-healing and recovery           | High         |

## Usage

```bash
# Apply a preset
helm preset apply engineering

# Customize a preset
helm preset apply engineering --override budget.daily_limit_cents=100000

# List available presets
helm preset list

# Export preset as bundle
helm preset export engineering -o ./my-engineering-bundle/
```
