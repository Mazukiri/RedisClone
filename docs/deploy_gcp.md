# Deploying MemKV to Google Cloud

This guide explains how to deploy your MemKV server to Google Cloud and verify it using the embedded dashboard.

## Prerequisites
- Google Cloud SDK (`gcloud`) installed.
- Docker installed.
- A Google Cloud Project ID.

## 1. Build and Test Locally

First, ensure the Docker image builds and the dashboard works.

```bash
# Build the image
docker build -t memkv-server .

# Run locally
docker run -p 8082:8082 -p 9090:9090 memkv-server
```

Open http://localhost:9090 in your browser. You should see the **MemKV Dashboard**.

## 2. Push to Google Container Registry (GCR)

Replace `YOUR_PROJECT_ID` with your actual GCP Project ID.

```bash
# Tag the image
docker tag memkv-server gcr.io/YOUR_PROJECT_ID/memkv-server

# Push to Registry
docker push gcr.io/YOUR_PROJECT_ID/memkv-server
```

## 3. Deploy to Cloud Run

Cloud Run is the easiest way to run containers. We need to expose port 8082 (Redis) and 9090 (Dashboard).
*Note: Cloud Run typically exposes only one port per service. For production, you might deploy to GKE or Compute Engine. For this demo, we can deploy to **Compute Engine** or use Cloud Run with sidecars if advanced config is OK. A simpler approach for valid testing is **Google Compute Engine (VM)**.*

### Option A: Compute Engine (Recommended for TCP + HTTP)

```bash
# Create a VM
gcloud compute instances create-with-container memkv-vm \
    --container-image gcr.io/YOUR_PROJECT_ID/memkv-server \
    --tags=memkv-server

# Open Ports in Firewall
gcloud compute firewall-rules create allow-memkv \
    --allow tcp:8082,tcp:9090 \
    --target-tags=memkv-server
```

**Result**: You will get an External IP.
- **Dashboard**: `http://<EXTERNAL-IP>:9090`
- **Redis**: `<EXTERNAL-IP>:8082`

## 4. Run Stress Tests against Cloud

Update your local `k6` script or `chaos.py` to point to the Cloud IP.

**Example (Chaos Test):**
```bash
# Edit internal/tests/chaos/chaos.py -> change HOST to '<EXTERNAL-IP>'
python3 memkv/tests/chaos/chaos.py
```

**Watch the Dashboard**:
While the test runs, open the Dashboard URL. You will see the **Connected Clients** spike and the **Memory Usage** grow in real-time!
