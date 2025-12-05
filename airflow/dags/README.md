<!-- @format -->

# Airflow DAGs for Prometheus ML Pipeline

Automated workflows for ML model training and deployment.

## DAGs

### regime_model_training.py

Automates regime detection model training pipeline.

**Schedule**: Weekly on Sunday at 2 AM UTC

**Workflow**:

```
1. check_data
   └─> Check ClickHouse for labeled samples (min 300)
   └─> Validate class distribution

2. train_model
   └─> Run training script with latest data
   └─> Export Random Forest to ONNX
   └─> Calculate accuracy metrics

3. validate_model (Branch)
   ├─> If accuracy >= 65% AND better than prod: → deploy_model
   └─> If accuracy < 65% OR not better: → skip_deployment

4. deploy_model
   └─> Backup current model
   └─> Copy new model to production
   └─> Save metrics metadata

5. notify_success
   └─> Send Telegram/Slack notification
```

**Requirements**:

- ClickHouse connection configured
- Python dependencies installed (see scripts/ml/regime/requirements.txt)
- Write access to models/ directory
- Notification service configured (Telegram bot token)

## Setup

### 1. Install Airflow (Docker)

```bash
# From project root
cd airflow
docker-compose up -d
```

### 2. Configure Connections

In Airflow UI (http://localhost:8080):

**ClickHouse Connection:**

- Connection ID: `clickhouse_default`
- Connection Type: `HTTP`
- Host: `clickhouse`
- Port: `9000`
- Login: (from ENV)
- Password: (from ENV)

### 3. Configure Variables

In Airflow UI → Admin → Variables:

- `MODELS_DIR`: `/app/models`
- `MIN_LABELED_SAMPLES`: `300`
- `MIN_ACCURACY`: `0.65`
- `TELEGRAM_BOT_TOKEN`: (your bot token)
- `TELEGRAM_ADMIN_CHAT_ID`: (admin chat ID)

### 4. Enable DAG

In Airflow UI → DAGs → Toggle `regime_model_training` to ON

## Monitoring

View DAG runs in Airflow UI:

- **DAG Runs**: Overall pipeline execution history
- **Task Instance**: Individual task logs and metrics
- **Gantt Chart**: Execution timeline
- **Graph View**: Task dependencies

## Manual Trigger

To trigger training manually (outside schedule):

```bash
airflow dags trigger regime_model_training
```

## Notifications

Success notification example:

```
🎯 Regime Detection Model Training Complete

✅ Status: SUCCESS
📊 Test Accuracy: 78.5%
📈 Training Samples: 450
🕐 Completed: 2024-11-28 02:15 UTC

Model deployed to production.
Application will use new model on next restart.
```

Failure notification example:

```
❌ Regime Detection Model Training Failed

Task: train_model
Error: Insufficient data quality
Samples: 250 (need 300+)

Please label more data and retry.
```

## Troubleshooting

**DAG not running**:

- Check scheduler is running: `airflow scheduler`
- Verify DAG is enabled in UI
- Check logs: `airflow tasks test regime_model_training check_data 2024-11-28`

**Training fails**:

- Check Python dependencies: `pip list | grep sklearn`
- Verify ClickHouse connection: `clickhouse-client --query "SELECT 1"`
- Check disk space in models/ directory

**Model not deploying**:

- Check accuracy threshold (MIN_ACCURACY variable)
- Compare with production metrics (models/regime_detector_metrics.json)
- Review validation logs in Airflow UI
