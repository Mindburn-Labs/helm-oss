# HELM Microsoft AutoGen Shim (Example)
import autogen
from helm_sdk import HelmClient

helm = HelmClient(base_url="http://localhost:8080/v1")

def governed_reply(recipient, messages, sender, config):
    # Intercept AutoGen reply and verify via HELM
    last_msg = messages[-1]["content"]
    verdict = helm.evaluate_intent(last_msg)
    if not verdict.allow:
        return True, f"Blocked by HELM: {verdict.reason_code}"
    return False, None

# Register the interceptor
user_proxy = autogen.UserProxyAgent("user_proxy")
user_proxy.register_reply(
    [autogen.Agent, None],
    reply_func=governed_reply,
    order=0
)
