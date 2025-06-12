import subprocess
import json
import yaml

def load_config(path="config.yaml"):
    print(f"📂 Loading config from: {path}")
    try:
        with open(path) as f:
            raw = f.read()
            print(f"📝 Raw YAML:\n{raw[:500]}")
            config = yaml.safe_load(raw)
            print(f"✅ Parsed config keys: {list(config.keys())}")
            return config["config"]
    except Exception as e:
        print(f"❌ Failed to load config.yaml: {e}")
        raise

def query_application(app_address: str, rpc: str):
    cmd = [
        "pocketd", "q", "application", "show-application",
        app_address, "--node", rpc, "--output", "json"
    ]
    try:
        result = subprocess.run(cmd, check=True, capture_output=True, text=True)
        return json.loads(result.stdout)["application"]
    except subprocess.CalledProcessError as e:
        return {"error": f"Subprocess failed: {e}"}
    except Exception as e:
        print(f"⚠️ JSON parse failed for {app_address}: {e}")
        print("Output:\n", result.stdout[:300])
        return {"error": "Parse failure"}
